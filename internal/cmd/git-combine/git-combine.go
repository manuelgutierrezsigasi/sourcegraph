package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

type parser struct {
	b *bufio.Reader

	cmd  []byte
	line []byte

	unreadCmd bool
}

func (p *parser) parseCmd() ([]byte, error) {
	if p.unreadCmd {
		p.unreadCmd = false
		return p.cmd, nil
	}

	line, err := p.b.ReadBytes('\n')
	if err != nil {
		if len(line) != 0 && err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}

	var cmd []byte
	if sp := bytes.IndexByte(line, ' '); sp < 0 {
		cmd = line[:len(line)-1] // trim off trailing \n
	} else {
		cmd = line[:sp]
	}

	p.cmd = cmd
	p.line = line
	return cmd, nil
}

// todoSkipOptionalCmd is a helper to simplify code around unimplemented commands. If
// the current command is not cmd then this is a noop. Otherwise we run and
// return parseCmd.
func (p *parser) todoSkipOptionalCmd(cmd string) ([]byte, error) {
	if string(p.cmd) != cmd {
		return p.cmd, nil
	}
	return p.parseCmd()
}

// todoSkipCmd is a helper to simplify code around unimplemented commands. If
// the current command is not cmd then an error is returned. Otherwise we run
// and return parseCmd.
func (p *parser) todoSkipCmd(cmd string) ([]byte, error) {
	if string(p.cmd) != cmd {
		return nil, fmt.Errorf("expected %q command, got %q", cmd, string(p.cmd))
	}
	return p.parseCmd()
}

// peekDiscard checks if the next bytes to read are equal to s. If so it
// discards them and returns true.
func (p *parser) peekDiscard(s string) (bool, error) {
	peek, err := p.b.Peek(len(s))
	if err != nil {
		return false, err
	}
	if string(peek) == s {
		if _, err := p.b.Discard(len(peek)); err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func (p *parser) parseData() error {
	// 'data' SP <arg> LF
	arg := p.line
	if sp := bytes.IndexByte(arg, ' '); sp < 0 {
		return fmt.Errorf("malformed data line: %q", string(arg))
	} else {
		arg = arg[sp+1 : len(arg)-1]
	}

	// Delimited format
	if bytes.HasPrefix(arg, []byte("<<")) {
		// 'data' SP '<<' <delim> LF
		// <raw> LF
		// <delim> LF
		// LF?
		delim := string(arg[2:]) + "\n"
		for {
			// TODO do something with raw
			if _, err := p.b.ReadBytes(delim[0]); err != nil {
				return err
			}
			// go back one byte so we can peek if we have delim
			if err := p.b.UnreadByte(); err != nil {
				return err
			}
			if found, err := p.peekDiscard(delim); err != nil {
				return err
			} else if found {
				break
			}
			// we unread a byte, so discard it now so we keep moving forward
			if _, err := p.b.Discard(1); err != nil {
				return err
			}
		}
		// Discard optional trailing LF
		if _, err := p.peekDiscard("\n"); err != nil {
			return err
		}

		return nil
	}

	// Exact byte count format
	// 'data' SP <count> LF
	// <raw> LF?
	count, err := strconv.Atoi(string(arg))
	if err != nil {
		return err
	}

	// TODO do something with raw
	if _, err := p.b.Discard(count); err != nil {
		return err
	}

	// Discard optional trailing LF
	if _, err := p.peekDiscard("\n"); err != nil {
		return err
	}

	return nil
}

func (p *parser) parseBlob() error {
	// 'blob' LF
	// mark?
	// original-oid?
	// data

	cmd, err := p.parseCmd()
	if err != nil {
		return err
	}
	cmd, err = p.todoSkipOptionalCmd("mark")
	if err != nil {
		return err
	}
	cmd, err = p.todoSkipOptionalCmd("original-oid")
	if err != nil {
		return err
	}
	if string(cmd) != "data" {
		return fmt.Errorf("expected data command in blob, got %q", string(cmd))
	}

	return p.parseData()
}

func (p *parser) parseCommit() error {
	// 'commit' SP <ref> LF
	// mark?
	// original-oid?
	// ('author' (SP <name>)? SP LT <email> GT SP <when> LF)?
	// 'committer' (SP <name>)? SP LT <email> GT SP <when> LF
	// ('encoding' SP <encoding>)?
	// data
	// ('from' SP <commit-ish> LF)?
	// ('merge' SP <commit-ish> LF)*
	// (filemodify | filedelete | filecopy | filerename | filedeleteall | notemodify)*
	// LF?

	cmd, err := p.parseCmd()
	if err != nil {
		return err
	}
	cmd, err = p.todoSkipOptionalCmd("mark")
	if err != nil {
		return err
	}
	cmd, err = p.todoSkipOptionalCmd("original-oid")
	if err != nil {
		return err
	}
	cmd, err = p.todoSkipOptionalCmd("author")
	if err != nil {
		return err
	}
	cmd, err = p.todoSkipCmd("committer")
	if err != nil {
		return err
	}
	cmd, err = p.todoSkipOptionalCmd("encoding")
	if err != nil {
		return err
	}
	if string(cmd) != "data" {
		return fmt.Errorf("expected data command in commit, got %q", string(cmd))
	}
	if err := p.parseData(); err != nil {
		return err
	}
	cmd, err = p.parseCmd()
	cmd, err = p.todoSkipOptionalCmd("from")
	if err != nil {
		return err
	}
	// TODO
	for string(cmd) == "merge" {
		cmd, err = p.parseCmd()
		if err != nil {
			return err
		}
	}
	for {
		done := false
		switch string(cmd) {
		case "M": // filemodify
			parts := bytes.SplitN(p.line, []byte{' '}, 4)
			if string(parts[2]) == "inline" {
				for _, v := range parts {
					fmt.Printf("%q\n", string(v))
				}
			}
		case "D": // filedelete
		case "C": // filecopy
		case "R": // filerename
		case "deleteall": // filedeleteall
		case "N": // notemodify
		default:
			done = true
		}
		if done {
			break
		}
		cmd, err = p.parseCmd()
	}

	// Discard optional trailing LF
	if _, err := p.peekDiscard("\n"); err != nil {
		return err
	}

	return nil
}

func (p *parser) parseReset() error {
	// 'reset' SP <ref> LF
	// ('from' SP <commit-ish> LF)?
	// LF?
	_, err := p.parseCmd()
	if err != nil {
		return err
	}
	_, err = p.todoSkipOptionalCmd("from")
	if err != nil {
		return err
	}
	// skip empty line
	_, err = p.todoSkipOptionalCmd("")
	if err != nil {
		return err
	}
	// we have gone 1 command too far
	p.unreadCmd = true
	return nil
}

func (p *parser) next() error {
	cmd, err := p.parseCmd()
	if err != nil {
		return err
	}

	switch string(cmd) {
	case "blob":
		return p.parseBlob()

	case "commit":
		return p.parseCommit()

	case "reset":
		return p.parseReset()

	default:
		return fmt.Errorf("unknown cmd %q: %q", string(cmd), string(p.line))
	}
}

func parse(r io.Reader) error {
	p := &parser{
		b: bufio.NewReader(r),
	}

	for {
		err := p.next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func main() {
	err := parse(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
}
