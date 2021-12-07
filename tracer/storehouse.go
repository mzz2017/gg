package tracer

type Storehouse map[int]map[int]interface{}

func MakeStorehouse() Storehouse {
	return make(Storehouse)
}

func (s Storehouse) Get(pid int, syscallNumber int) (v interface{}, ok bool) {
	if _, ok := s[pid]; !ok {
		return nil, false
	}
	m, ok := s[pid][syscallNumber]
	if !ok {
		return nil, false
	}
	return m, true
}

func (t Storehouse) Save(pid int, syscallNumber int, v interface{}) {
	if t[pid] == nil {
		t[pid] = make(map[int]interface{})
	}
	t[pid][syscallNumber] = v
}

func (t Storehouse) Remove(pid int, syscallNumber int) {
	if _, ok := t[pid]; !ok {
		return
	}
	if _, ok := t[pid][syscallNumber]; ok {
		delete(t[pid], syscallNumber)
		if len(t[pid]) == 0 {
			delete(t, pid)
		}
	}
}

func (t Storehouse) RemoveAll(pid int) {
	if _, ok := t[pid]; !ok {
		return
	}
	if len(t[pid]) == 0 {
		delete(t, pid)
	}
}
