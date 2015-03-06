package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("Usage: %s <application> <command> <environment>", os.Args[0])
	}

	application := os.Args[1]
	host := fmt.Sprint(application, ".nine.ch:22")
	user := os.Args[1]
	environment := os.Args[3]

	command := ""
	switch os.Args[2] {
	case "t", "tail":
		command = fmt.Sprint("cd ", application, "/current && tail -f log/", environment, ".log")
	case "c", "console":
		command = fmt.Sprint("cd ", application, "/current && bundle exec rails console ", environment)
	case "test":
		command = fmt.Sprint("echo test")
	}

	if command == "" {
		panic(fmt.Sprint("unknown switch ", os.Args[2]))
	}

	client, session, err := connectToHost(user, host)
	if err != nil {
		panic(err)
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		panic(err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}

	if err := session.Run(command); err != nil {
		panic("Failed to run: " + err.Error())
	}
	client.Close()
}

func connectToHost(user, host string) (*ssh.Client, *ssh.Session, error) {
	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatal(err)
	}
	agent := agent.NewClient(sock)

	signers, err := agent.Signers()
	if err != nil {
		log.Fatal(err)
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signers...)},
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}
