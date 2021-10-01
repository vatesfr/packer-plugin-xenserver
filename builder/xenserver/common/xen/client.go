package xen

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"

	xenapi "github.com/terra-farm/go-xen-api-client"
)

func Unpause(c *Connection, vmRef xenapi.VMRef) (err error) {
	err = c.client.VM.Unpause(c.session, vmRef)
	if err != nil {
		return err
	}
	return
}

func GetDisks(c *Connection, vmRef xenapi.VMRef) (vdis []xenapi.VDIRef, err error) {
	// Return just data disks (non-isos)
	vdis = make([]xenapi.VDIRef, 0)
	vbds, err := c.client.VM.GetVBDs(c.session, vmRef)
	if err != nil {
		return nil, err
	}

	for _, vbd := range vbds {
		rec, err := c.client.VBD.GetRecord(c.session, vbd)
		if err != nil {
			return nil, err
		}
		if rec.Type == "Disk" {

			vdi, err := c.client.VBD.GetVDI(c.session, vbd)
			if err != nil {
				return nil, err
			}
			vdis = append(vdis, vdi)

		}
	}
	return vdis, nil
}

func ConnectVdi(c *Connection, vmRef xenapi.VMRef, vdiRef xenapi.VDIRef, vbdType xenapi.VbdType) (err error) {

	var mode xenapi.VbdMode
	var unpluggable bool
	var bootable bool
	var t xenapi.VbdType
	switch vbdType {
	case xenapi.VbdTypeCD:
		mode = xenapi.VbdModeRO
		bootable = true
		unpluggable = false
		t = xenapi.VbdTypeCD
	case xenapi.VbdTypeDisk:
		mode = xenapi.VbdModeRW
		bootable = false
		unpluggable = false
		t = xenapi.VbdTypeDisk
	case xenapi.VbdTypeFloppy:
		mode = xenapi.VbdModeRW
		bootable = false
		unpluggable = true
		t = xenapi.VbdTypeFloppy
	}

	vbd_ref, err := c.client.VBD.Create(c.session, xenapi.VBDRecord{
		VM:         xenapi.VMRef(vmRef),
		VDI:        xenapi.VDIRef(vdiRef),
		Userdevice: "autodetect",
		Empty:      false,
		// OtherConfig: map[string]interface{{}},
		QosAlgorithmType: "",
		// QosAlgorithmParams: map[string]interface{{}},
		Mode:        mode,
		Unpluggable: unpluggable,
		Bootable:    bootable,
		Type:        t,
	})

	if err != nil {
		return err
	}

	fmt.Println("VBD Ref:", vbd_ref)

	uuid, err := c.client.VBD.GetUUID(c.session, vbd_ref)

	fmt.Println("VBD UUID: ", uuid)
	/*
	   // 2. Plug VBD (Non need - the VM hasn't booted.
	   // @todo - check VM state
	   result = APIResult{}
	   err = self.Client.APICall(&result, "VBD.plug", vbd_ref)

	   if err != nil {
	       return err
	   }
	*/
	return
}

func DisconnectVdi(c *Connection, vmRef xenapi.VMRef, vdi xenapi.VDIRef) error {
	vbds, err := c.client.VM.GetVBDs(c.session, vmRef)
	if err != nil {
		return fmt.Errorf("Unable to get VM VBDs: %s", err.Error())
	}

	for _, vbd := range vbds {
		rec, err := c.client.VBD.GetRecord(c.session, vbd)
		if err != nil {
			return fmt.Errorf("Could not get record for VBD '%s': %s", vbd, err.Error())
		}
		recVdi := rec.VDI
		if recVdi == vdi {
			_ = c.client.VBD.Unplug(c.session, vbd)
			err = c.client.VBD.Destroy(c.session, vbd)
			if err != nil {
				return fmt.Errorf("Could not destroy VBD '%s': %s", vbd, err.Error())
			}

			return nil
		} else {
			log.Printf("Could not find VDI record in VBD '%s'", vbd)
		}
	}

	return fmt.Errorf("Could not find VBD for VDI '%s'", vdi)
}

func ConnectNetwork(c *Connection, networkRef xenapi.NetworkRef, vmRef xenapi.VMRef, device string) (*xenapi.VIFRef, error) {
	vif, err := c.client.VIF.Create(c.session, xenapi.VIFRecord{
		Network:     networkRef,
		VM:          vmRef,
		Device:      device,
		LockingMode: xenapi.VifLockingModeNetworkDefault,
	})

	if err != nil {
		return nil, err
	}
	log.Printf("Created the following VIF: %s", vif)

	return &vif, nil
}

// VDI associated functions
// Expose a VDI using the Transfer VM
// (Legacy VHD export)

type TransferRecord struct {
	UrlFull string `xml:"url_full,attr"`
}

func Expose(c *Connection, vdiRef xenapi.VDIRef, format string) (url string, err error) {

	hosts, err := c.client.Host.GetAll(c.session)

	if err != nil {
		err = errors.New(fmt.Sprintf("Could not retrieve hosts in the pool: %s", err.Error()))
		return "", err
	}
	host := hosts[0]

	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to get VDI uuid for %s: %s", vdiRef, err.Error()))
		return "", err
	}

	args := make(map[string]string)
	args["transfer_mode"] = "http"
	args["vdi_uuid"] = string(vdiRef)
	args["expose_vhd"] = "true"
	args["network_uuid"] = "management"
	args["timeout_minutes"] = "5"

	handle, err := c.client.Host.CallPlugin(c.session, host, "transfer", "expose", args)

	if err != nil {
		err = errors.New(fmt.Sprintf("Error whilst exposing VDI %s: %s", vdiRef, err.Error()))
		return "", err
	}

	args = make(map[string]string)
	args["record_handle"] = handle
	record_xml, err := c.client.Host.CallPlugin(c.session, host, "transfer", "get_record", args)

	if err != nil {
		err = errors.New(fmt.Sprintf("Unable to retrieve transfer record for VDI %s: %s", vdiRef, err.Error()))
		return "", err
	}

	var record TransferRecord
	xml.Unmarshal([]byte(record_xml), &record)

	if record.UrlFull == "" {
		return "", errors.New(fmt.Sprintf("Error: did not parse XML properly: '%s'", record_xml))
	}

	// Handles either raw or VHD formats

	switch format {
	case "vhd":
		url = fmt.Sprintf("%s.vhd", record.UrlFull)

	case "raw":
		url = record.UrlFull
	}

	return
}

func Unexpose(c *Connection, vdiRef xenapi.VDIRef) (err error) {

	disk_uuid, err := c.client.VDI.GetUUID(c.session, vdiRef)

	if err != nil {
		return err
	}

	hosts, err := c.client.Host.GetAll(c.session)

	if err != nil {
		err = errors.New(fmt.Sprintf("Could not retrieve hosts in the pool: %s", err.Error()))
		return err
	}

	host := hosts[0]

	args := make(map[string]string)
	args["vdi_uuid"] = disk_uuid

	result, err := c.client.Host.CallPlugin(c.session, host, "transfer", "unexpose", args)

	if err != nil {
		return err
	}

	log.Println(fmt.Sprintf("Unexpose result: %s", result))

	return nil
}

// Client Initiator
type Connection struct {
	client   *xenapi.Client
	session  xenapi.SessionRef
	Host     string
	Port     int
	Username string
	Password string
}

func (c Connection) GetSession() string {
	return string(c.session)
}

func NewXenAPIClient(host string, port int, username, password string) (*Connection, error) {
	url := fmt.Sprintf("https://%s/", net.JoinHostPort(host, strconv.Itoa(port)))
	client, err := xenapi.NewClient(url, nil)
	if err != nil {
		return nil, err
	}

	session, err := client.Session.LoginWithPassword(username, password, "1.0", "packer")
	if err != nil {
		return nil, err
	}

	return &Connection{client, session, host, port, username, password}, nil
}

func (c *Connection) GetClient() *xenapi.Client {
	return c.client
}

func (c *Connection) GetSessionRef() xenapi.SessionRef {
	return c.session
}
