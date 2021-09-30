package common

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/mitchellh/go-vnc"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type VNCConnectionWrapper struct {
	TlsConn *tls.Conn
	rawConn net.Conn
}

func (v VNCConnectionWrapper) Close() {
	if v.TlsConn != nil {
		_ = v.TlsConn.Close()
	}

	if v.rawConn != nil {
		_ = v.rawConn.Close()
	}
}

type VNCClientWrapper struct {
	Client *vnc.ClientConn

	connection *VNCConnectionWrapper
}

func (v VNCClientWrapper) Close() {
	if v.Client != nil {
		_ = v.Client.Close()
	}

	v.connection.Close()
}

func CreateVNCConnection(state multistep.StateBag, location string) (*VNCConnectionWrapper, error) {
	xenClient := state.Get("client").(*Connection)
	wrapper := VNCConnectionWrapper{}

	var err error

	target, err := getTcpAddress(location)
	if err != nil {
		return nil, err
	}

	wrapper.rawConn, err = ConnectViaXenProxy(state, target)
	if err != nil {
		return nil, err
	}

	wrapper.TlsConn, err = httpConnectRequest(location, string(xenClient.GetSessionRef()), wrapper.rawConn)
	if err != nil {
		wrapper.Close()
		return nil, err
	}

	return &wrapper, nil
}

func ConnectVNC(state multistep.StateBag, location string) (*VNCClientWrapper, error) {
	wrapper := VNCClientWrapper{}

	var err error

	wrapper.connection, err = CreateVNCConnection(state, location)
	if err != nil {
		return nil, err
	}

	wrapper.Client, err = vnc.Client(wrapper.connection.TlsConn, &vnc.ClientConfig{
		Exclusive: false,
	})
	if err != nil {
		wrapper.Close()
		return nil, err
	}

	return &wrapper, nil
}

func httpConnectRequest(location string, xenSessionRef string, proxyConnection net.Conn) (*tls.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	tlsConnection := tls.Client(proxyConnection, tlsConfig)

	request, err := http.NewRequest(http.MethodConnect, location, http.NoBody)

	if err != nil {
		return nil, fmt.Errorf("could not connect to xen console api: %w", err)
	}

	request.Close = false
	request.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: xenSessionRef,
	})

	err = request.Write(tlsConnection)
	if err != nil {
		return nil, fmt.Errorf("could not write to tls connection: %w", err)
	}

	// Look for \r\n\r\n sequence. Everything after the HTTP Header is for the vnc client.
	response, err := readHttpHeader(tlsConnection)
	if err != nil {
		return nil, fmt.Errorf("could not read http header: %w", err)
	}

	log.Printf("Received response: %s", response)
	return tlsConnection, nil
}

func getTcpAddress(location string) (string, error) {
	parsedUrl, err := url.Parse(location)
	if err != nil {
		return "", err
	}

	var target string
	if parsedUrl.Port() != "" {
		target = parsedUrl.Host
	} else if parsedUrl.Scheme == "http" {
		target = parsedUrl.Host + ":80"
	} else if parsedUrl.Scheme == "https" {
		target = parsedUrl.Host + ":443"
	} else {
		return "", errors.New("unsupported protocol")
	}
	return target, nil
}

func readHttpHeader(tlsConn *tls.Conn) (string, error) {
	builder := strings.Builder{}
	buffer := make([]byte, 1)
	sequenceProgress := 0

	for {
		if _, err := io.ReadFull(tlsConn, buffer); err != nil {
			return "", fmt.Errorf("failed to start vnc session: %w", err)
		}

		builder.WriteByte(buffer[0])

		if buffer[0] == '\n' && sequenceProgress%2 == 1 {
			sequenceProgress++
		} else if buffer[0] == '\r' && sequenceProgress%2 == 0 {
			sequenceProgress++
		} else {
			sequenceProgress = 0
		}

		if sequenceProgress == 4 {
			break
		}
	}
	return builder.String(), nil
}
