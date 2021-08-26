package netstack

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"inet.af/netaddr"
	"tailscale.com/net/tsaddr"
)

func doSSHDemoThing(c net.Conn) error {
	hostKey, err := ioutil.ReadFile("/etc/ssh/ssh_host_ed25519_key")
	if err != nil {
		return err
	}
	signer, err := gossh.ParsePrivateKey(hostKey)
	if err != nil {
		return err
	}
	srv := &ssh.Server{
		Handler:           handleSSH,
		RequestHandlers:   map[string]ssh.RequestHandler{},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{},
		ChannelHandlers:   map[string]ssh.ChannelHandler{},
	}
	for k, v := range ssh.DefaultRequestHandlers {
		srv.RequestHandlers[k] = v
	}
	for k, v := range ssh.DefaultChannelHandlers {
		srv.ChannelHandlers[k] = v
	}
	for k, v := range ssh.DefaultSubsystemHandlers {
		srv.SubsystemHandlers[k] = v
	}
	srv.AddHostKey(signer)

	srv.HandleConn(c)
	return nil
}

func handleSSH(s ssh.Session) {
	user := s.User()
	addr := s.RemoteAddr()
	log.Printf("Handling SSH from %v for user %v", addr, user)
	ta, ok := addr.(*net.TCPAddr)
	if !ok {
		log.Printf("tsshd: rejecting non-TCP addr %T %v", addr, addr)
		s.Exit(1)
		return
	}
	tanetaddr, ok := netaddr.FromStdIP(ta.IP)
	if !ok {
		log.Printf("tsshd: rejecting unparseable addr %v", ta.IP)
		s.Exit(1)
		return
	}
	if !tsaddr.IsTailscaleIP(tanetaddr) {
		log.Printf("tsshd: rejecting non-Tailscale addr %v", ta.IP)
		s.Exit(1)
		return
	}

	ptyReq, winCh, isPty := s.Pty()
	if !isPty {
		fmt.Fprintf(s, "TODO scp etc")
		s.Exit(1)
		return
	}
	_ = winCh

	fmt.Fprintf(s, "Hello, %v from %v. You wanted %+v\n", user, addr, ptyReq)
	s.Exit(0)
	return
}
