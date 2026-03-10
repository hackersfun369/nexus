package chat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// REPL runs the interactive read-eval-print loop
type REPL struct {
	session *Session
	in      io.Reader
	out     io.Writer
}

// NewREPL creates a new REPL with the given session
func NewREPL(sess *Session) *REPL {
	return &REPL{
		session: sess,
		in:      os.Stdin,
		out:     os.Stdout,
	}
}

// NewREPLWithIO creates a REPL with custom IO (for testing)
func NewREPLWithIO(sess *Session, in io.Reader, out io.Writer) *REPL {
	return &REPL{session: sess, in: in, out: out}
}

// Run starts the REPL loop
func (r *REPL) Run(ctx context.Context) {
	r.printBanner()
	scanner := bufio.NewScanner(r.in)
	for {
		fmt.Fprintf(r.out, "\n%s%s%s",
			colorBold+colorBlue, r.session.Prompt(), colorReset)

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		cmd := ParseCommand(input)
		response, quit := Handle(ctx, cmd, r.session)

		if response != "" {
			fmt.Fprintln(r.out, response)
		}

		r.session.AddHistory(input, response)

		if quit {
			break
		}
	}
}

func (r *REPL) printBanner() {
	banner := colorBold + colorCyan + `
  ███╗   ██╗███████╗██╗  ██╗██╗   ██╗███████╗
  ████╗  ██║██╔════╝╚██╗██╔╝██║   ██║██╔════╝
  ██╔██╗ ██║█████╗   ╚███╔╝ ██║   ██║███████╗
  ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║╚════██║
  ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝███████║
  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝` + colorReset

	fmt.Fprintln(r.out, banner)
	fmt.Fprintln(r.out, colorDim+"  intelligent development system — symbolic AI edition"+colorReset)
	fmt.Fprintln(r.out, colorDim+"  Type 'help' for commands or 'quit' to exit"+colorReset)
}
