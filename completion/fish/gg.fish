# fish completion for gg

function __fish_gg_print_remaining_args
    set -l tokens (commandline -opc) (commandline -ct)
    set -e tokens[1]
    if test -n "$argv"
        and not string match -qr '^-' $argv[1]
        string join0 -- $argv
        return 0
    else
        return 1
    end
end

function __fish_complete_gg_subcommand
    set -l args (__fish_gg_print_remaining_args | string split0)
    __fish_complete_subcommand --commandline $args
end

# Complete the command we are executed under gg
complete -c gg -x -a "(__fish_complete_gg_subcommand)"
