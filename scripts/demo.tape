# The terminal recording the README embeds under "Command-line tool": check
# and create on the ugent/basic example, then the package directory. Run from
# the repository root with `vhs scripts/demo.tape` whenever the console output
# of check or create, or the package layout, changes. Needs vhs and tree
# (CONTRIBUTING.md, Requirements).

Output docs/demo.gif

Require go
Require tree

Set FontSize 12
Set Width 660
Set Height 340
Set Padding 20
Set Framerate 20
Set TypingSpeed 40ms

# Build, configure the submitter and clear the screen out of frame, so the
# recording starts at the first command.
Hide
Type "go build -o bin/sip-creator . && export SIP_SUBMITTER_NAME='Example Organization' && rm -rf tmp/demo && clear"
Enter
Sleep 8s
Show

Type "./bin/sip-creator check --profile ugent/basic examples/ugent/basic"
Sleep 500ms
Enter
Sleep 3s

# Each command starts on a cleared screen, so its output stands alone in the
# frame and the terminal stays small.
Hide
Type "clear"
Enter
Show

Type "./bin/sip-creator create --profile ugent/basic examples/ugent/basic tmp/demo"
Sleep 500ms
Enter
Sleep 3s

Hide
Type "clear"
Enter
Show

# The resting frame: the package directory and its zip. The README's text
# shows the tree three levels deep.
Type "tree -L 2 --noreport tmp/demo"
Sleep 500ms
Enter
Sleep 6s
