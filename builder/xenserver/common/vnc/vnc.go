package vnc

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/mitchellh/go-vnc"
	config2 "github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/config"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/proxy"
	"github.com/xenserver/packer-builder-xenserver/builder/xenserver/common/xen"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func GetVNCConsoleLocation(state multistep.StateBag) (string, error) {
	xenClient := state.Get("client").(*xen.Connection)
	config := state.Get("commonconfig").(config2.CommonConfig)

	vmRef, err := xenClient.GetClient().VM.GetByNameLabel(xenClient.GetSessionRef(), config.VMName)

	if err != nil {
		return "", err
	}

	if len(vmRef) != 1 {
		return "", fmt.Errorf("expected to find a single VM, instead found '%d'. Ensure the VM name is unique", len(vmRef))
	}

	consoles, err := xenClient.GetClient().VM.GetConsoles(xenClient.GetSessionRef(), vmRef[0])

	if err != nil {
		return "", err
	}

	if len(consoles) != 1 {
		return "", fmt.Errorf("expected to find a VM console, instead found '%d'. Ensure there is only one console", len(consoles))
	}

	location, err := xenClient.GetClient().Console.GetLocation(xenClient.GetSessionRef(), consoles[0])

	if err != nil {
		return "", err
	}

	return location, nil
}

func CreateVNCConnection(state multistep.StateBag, location string) (net.Conn, error) {
	xenClient := state.Get("client").(*xen.Connection)
	xenProxy := state.Get("xen_proxy").(proxy.XenProxy)

	target, err := GetTcpAddressFromURL(location)
	if err != nil {
		return nil, err
	}

	rawConn, err := xenProxy.ConnectWithAddr(target)
	if err != nil {
		return nil, err
	}

	tlsConn, err := InitializeVNCConnection(location, string(xenClient.GetSessionRef()), rawConn)
	if err != nil {
		rawConn.Close()
		return nil, err
	}

	return tlsConn, nil
}

func CreateVNCClient(state multistep.StateBag, location string) (*vnc.ClientConn, error) {
	var err error

	connection, err := CreateVNCConnection(state, location)
	if err != nil {
		return nil, err
	}

	client, err := vnc.Client(connection, &vnc.ClientConfig{
		Exclusive: true,
	})
	if err != nil {
		connection.Close()
		return nil, err
	}

	return client, nil
}

func InitializeVNCConnection(location string, xenSessionRef string, rawConn net.Conn) (*tls.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	tlsConnection := tls.Client(rawConn, tlsConfig)

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

func GetTcpAddressFromURL(location string) (string, error) {
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
