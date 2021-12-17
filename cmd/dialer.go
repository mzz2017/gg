package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/1lann/promptui"
	"github.com/AlecAivazis/survey/v2"
	"github.com/mzz2017/gg/common"
	"github.com/mzz2017/gg/config"
	"github.com/mzz2017/gg/dialer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/tools/container/intsets"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var UnableToConnectErr = fmt.Errorf("unable to connect to the proxy node")

type DialerWithLatency struct {
	Dialer  *dialer.Dialer
	Latency int
}

func GetDialer(log *logrus.Logger) (d *dialer.Dialer, err error) {
	nodeLink := config.ParamsObj.Node
	if len(nodeLink) > 0 {
		d, err = GetDialerFromLink(nodeLink, config.ParamsObj.TestNode)
		if err != nil {
			return nil, err
		}
		return d, nil
	}
	if config.ParamsObj.Subscription.Link != "" {
		if d, err = GetDialerFromSubscription(log, config.ParamsObj.TestNode); err != nil {
			return nil, err
		}
		return d, nil
	}
	if d, err = GetDialerFromInput(config.ParamsObj.TestNode); err != nil {
		return nil, err
	}
	return d, nil
}

func GetDialerFromLink(nodeLink string, testNode bool) (d *dialer.Dialer, err error) {
	u, err := url.Parse(nodeLink)
	if err != nil {
		return nil, err
	}
	d, err = dialer.NewFromLink(u.Scheme, u.String())
	if err != nil {
		return nil, err
	}
	if testNode {
		if ok, err := d.Test(context.Background()); !ok {
			return nil, fmt.Errorf("%w: %v", UnableToConnectErr, err)
		}
	}
	return d, nil
}

func GetDialerFromInput(testNode bool) (d *dialer.Dialer, err error) {
	var link string
	// FIXME: Is it really necessary to introduce another one library?
	err = survey.AskOne(&survey.Input{
		Message: "Enter the share-link of your proxy:",
	}, &link, common.SetRequire)
	if err != nil {
		return nil, err
	}
	return GetDialerFromLink(strings.TrimSpace(link), testNode)
}

func GetDialerFromSubscription(log *logrus.Logger, testNode bool) (d *dialer.Dialer, err error) {
	if config.ParamsObj.Subscription.Link == "" {
		return nil, fmt.Errorf("subscription link is not set")
	}
	switch config.ParamsObj.Subscription.Select {
	case "manual", "select", "__select__":
		if config.ParamsObj.Subscription.CacheLastNode {
			if config.ParamsObj.Subscription.Select != "__select__" {
				d = GetDialerFromSubscriptionLastNodeCache(testNode)
				if d != nil {
					log.Infof("Use the cached node: %v\n", d.Name())
					return d, nil
				}
			}
			defer func() {
				if d != nil {
					_ = cacheSubscriptionNode(log, d)
				}
			}()
		}
		log.Infoln("Pulling the subscription...")
		dialers, err := pullDialersFromSubscription(log, config.ParamsObj.Subscription.Link)
		if err != nil {
			return nil, err
		}
		var result []*DialerWithLatency
		if testNode {
			log.Warnln("Test nodes...")
			result = testLatencies(log, dialers)
		} else {
			result = make([]*DialerWithLatency, 0, len(dialers))
			for i := range dialers {
				result = append(result, &DialerWithLatency{
					Dialer:  dialers[i],
					Latency: 0,
				})
			}
		}
		if len(result) == 0 {
			break
		}
		d, err := selectNodeFromInput(result)
		if err != nil {
			return nil, err
		}
		return d.Dialer, nil
	default:
		log.Warnf("Unexpected select option: %v. Fallback to \"first\".", config.ParamsObj.Subscription.Select)
		fallthrough
	case "first":
		if config.ParamsObj.Subscription.CacheLastNode {
			d = GetDialerFromSubscriptionLastNodeCache(testNode)
			if d != nil {
				log.Infof("Use the cached node: %v\n", d.Name())
				return d, nil
			}
			defer func() {
				if d != nil {
					_ = cacheSubscriptionNode(log, d)
				}
			}()
		}
		log.Infoln("Pulling the subscription...")
		dialers, err := pullDialersFromSubscription(log, config.ParamsObj.Subscription.Link)
		if err != nil {
			return nil, err
		}
		if testNode {
			log.Infoln("Finding the first available node...")
			if d = firstAvailableDialer(log, dialers); d != nil {
				log.Infof("Use the node: %v\n", d.Name())
				return d, nil
			}
		} else {
			if len(dialers) > 0 {
				log.Infof("Use the node: %v\n", dialers[0].Name())
				return dialers[0], nil
			}
		}
	}
	return nil, fmt.Errorf("cannot find any available node in your subscription, and you can try again with argument '-vv' to get more information")
}

func cacheSubscriptionNode(log *logrus.Logger, d *dialer.Dialer) error {
	v, configPath := getConfig(log, false, viper.New, nil)
	m := v.AllSettings()
	if v.GetString("subscription.link") == "" {
		// do not cache if config: "cache.subscription.link" is empty
		log.Infoln("did not cache the node because config: \"cache.subscription.link\" was empty")
		return nil
	}
	log.Infof("cache the node: %v: %v\n", d.Name(), d.Link())
	if err := config.SetValueHierarchicalMap(m, completeKey("cache.subscription.last_node"), d.Link()); err != nil {
		return err
	}
	return WriteConfig(m, configPath)
}

func firstAvailableDialer(log *logrus.Logger, dialers []*dialer.Dialer) *dialer.Dialer {
	concurrency := make(chan struct{}, 8)
	result := make(chan *dialer.Dialer, cap(concurrency))
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, d := range dialers {
		wg.Add(1)
		go func(d *dialer.Dialer) {
			defer func() { wg.Done() }()
			select {
			case <-ctx.Done():
				return
			case concurrency <- struct{}{}:
				defer func() {
					<-concurrency
				}()
				if ok, err := d.Test(ctx); ok {
					log.Tracef("test pass: %v", d.Name())
					cancel()
					result <- d
				} else if !errors.Is(err, context.Canceled) {
					log.Tracef("test fail: %v: %v", d.Name(), err)
				}
			}
		}(d)
	}
	wg.Wait()
	if len(result) > 0 {
		return <-result
	}
	return nil
}

func testLatencies(log *logrus.Logger, dialers []*dialer.Dialer) (result []*DialerWithLatency) {
	concurrency := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := range dialers {
		wg.Add(1)
		go func(i int) {
			d := dialers[i]
			concurrency <- struct{}{}
			defer func() {
				wg.Done()
				<-concurrency
			}()
			t := time.Now()
			b, _ := d.Test(context.Background())
			latency := int(time.Since(t).Milliseconds())
			if !b {
				latency = -1
			}
			mu.Lock()
			result = append(result, &DialerWithLatency{
				Dialer:  d,
				Latency: latency,
			})
			if len(result)%10 == 0 && len(result) != len(dialers) {
				log.Infof("Test nodes: %v/%v", len(result), len(dialers))
			}
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	log.Infof("Test nodes: %v/%v", len(result), len(dialers))
	return result
}

func GetDialerFromSubscriptionLastNodeCache(testNode bool) (d *dialer.Dialer) {
	if config.ParamsObj.Cache.Subscription.LastNode != "" {
		d, _ := GetDialerFromLink(config.ParamsObj.Cache.Subscription.LastNode, testNode)
		if d != nil {
			return d
		}
	}
	return nil
}

func selectNodeFromInput(nodes []*DialerWithLatency) (*DialerWithLatency, error) {
	sort.Slice(nodes, func(i, j int) bool {
		vi := nodes[i].Latency
		vj := nodes[j].Latency
		if vi == -1 {
			vi = intsets.MaxInt
		}
		if vj == -1 {
			vj = intsets.MaxInt
		}
		return vi < vj
	})
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "🛪 {{ .Dialer.Name | cyan }} ({{ .Latency | red }} ms)",
		Inactive: "  {{ .Dialer.Name | cyan }} ({{ .Latency | red }} ms)",
		Selected: "🛪 {{ .Dialer.Name | red | cyan }}",
		Details: `
--------- Detail ----------
{{ "Name:" | faint }}	{{ .Dialer.Name }}
{{ "Protocol:" | faint }}	{{ .Dialer.Protocol }}
{{ "Support UDP:" | faint }}	{{ .Dialer.SupportUDP }}
{{ "Latency:" | faint }}	{{ .Latency }} ms`,
	}
	searcher := func(input string, index int) bool {
		node := nodes[index]
		name := strings.Replace(strings.ToLower(node.Dialer.Name()), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}
	prompt := promptui.Select{
		Label:     "Select Node",
		Items:     nodes,
		Templates: templates,
		Size:      8,
		Searcher:  searcher,
	}
	i, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return nodes[i], nil
}
