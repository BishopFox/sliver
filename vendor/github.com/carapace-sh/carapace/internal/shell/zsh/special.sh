#compdef special
function _special_completion {
local -a subcmds
subcmds=('p1 & < > '"'"' " { } $ # | ? ( ) ;  [ ] * \\ $() ${} ` ``:description for c command' 'd:description for d command')
_describe 'command' subcmds
}
compquote '' 2>/dev/null && _special_completion
compdef _special_completion special
