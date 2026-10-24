// Package cli contains the subcommands of the binary.
package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"inventar/internal/auth"
	"inventar/internal/config"
	"inventar/internal/store"
)

const usage = `Usage: inventar <command> [options]

Commands:
  serve                                     Start the server
  create-user --name NAME --email EMAIL     Create a user (asks for the password)
  reset-password --email EMAIL              Set a new password and end all sessions

Configuration via environment variables, see README.
`

// command bundles what every subcommand needs.
type command struct {
	cfg    config.Config
	args   []string
	stdin  io.Reader
	stdout io.Writer
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return errors.New("no command given")
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		return err
	}
	c := command{cfg: cfg, args: args[1:], stdin: stdin, stdout: stdout}
	switch args[0] {
	case "serve":
		return serve(ctx, c)
	case "create-user":
		return createUser(ctx, c)
	case "reset-password":
		return resetPassword(ctx, c)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		fmt.Fprint(stdout, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func createUser(ctx context.Context, c command) error {
	var name, email string
	err := c.parseFlags("create-user", func(f *flag.FlagSet) {
		f.StringVar(&name, "name", "", "name of the person")
		f.StringVar(&email, "email", "", "email address for logging in")
	}, &name, &email)
	if err != nil {
		return err
	}
	return c.withPasswordAndStore(ctx, func(st *store.Store, hash string) error {
		u, err := st.CreateUser(ctx, strings.TrimSpace(name), email, hash, time.Now())
		if errors.Is(err, store.ErrEmailTaken) {
			return fmt.Errorf("the email %s is already taken", store.NormalizeEmail(email))
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "User %s <%s> created.\n", u.Name, u.Email)
		return nil
	})
}

func resetPassword(ctx context.Context, c command) error {
	var email string
	err := c.parseFlags("reset-password", func(f *flag.FlagSet) {
		f.StringVar(&email, "email", "", "email address of the user")
	}, &email)
	if err != nil {
		return err
	}
	return c.withPasswordAndStore(ctx, func(st *store.Store, hash string) error {
		err := st.SetPassword(ctx, email, hash)
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("no user with the email %s", store.NormalizeEmail(email))
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(c.stdout, "Password set, all sessions have been ended.")
		return nil
	})
}

// parseFlags reads the options of a subcommand; all required values passed must be set.
func (c command) parseFlags(name string, define func(*flag.FlagSet), required ...*string) error {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(c.stdout)
	define(flags)
	if err := flags.Parse(c.args); err != nil {
		return err
	}
	for _, r := range required {
		if strings.TrimSpace(*r) == "" {
			flags.Usage()
			return errors.New("required option missing")
		}
	}
	return nil
}

// withPasswordAndStore reads the password, opens the store and calls fn with the password hash.
func (c command) withPasswordAndStore(ctx context.Context, fn func(st *store.Store, hash string) error) error {
	hash, err := readPasswordHash(c.stdin, c.stdout)
	if err != nil {
		return err
	}
	st, err := store.Open(ctx, c.cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()
	return fn(st, hash)
}

// readPasswordHash reads the password hidden from the terminal or line by line from stdin.
func readPasswordHash(stdin io.Reader, stdout io.Writer) (string, error) {
	fmt.Fprint(stdout, "Password: ")
	var password string
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		input, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(stdout)
		if err != nil {
			return "", err
		}
		password = string(input)
	} else {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		password = strings.TrimRight(line, "\r\n")
	}
	return auth.HashPassword(password)
}
