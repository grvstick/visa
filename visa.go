// Copyright (c) 2015-2020 The usbtmc developers. All rights reserved.
// Project site: https://github.com/gotmc/usbtmc
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package visa

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/gousb"
	"github.com/grvstick/usbtmc"
)

// VisaResource represents a VISA enabled piece of test equipment.

type VisaResource struct {
	resourceString string
	interfaceType  string
	boardIndex     int
	manufacturerID int
	modelCode      int
	serialNumber   string
	interfaceIndex int
	resourceClass  string
}

// parseVisaResource creates a new VisaResource using the given VISA resourceString.
func parseVisaResource(resourceString string) (*VisaResource, error) {
	visa := &VisaResource{
		resourceString: resourceString,
		interfaceType:  "",
		boardIndex:     -1,
		manufacturerID: -1,
		modelCode:      -1,
		serialNumber:   "",
		interfaceIndex: -1,
		resourceClass:  "",
	}
	regString := `^(?P<interfaceType>[A-Za-z]+)(?P<boardIndex>\d*)::` +
		`(?P<manufacturerID>[^\s:]+)::` +
		`(?P<modelCode>[^\s:]+)` +
		`(::(?P<serialNumber>[^\s:]+))?` +
		`(::(?P<interfaceIndex>\d*))` +
		`::(?P<resourceClass>[^\s:]+)$`

	re := regexp.MustCompile(regString)
	res := re.FindStringSubmatch(resourceString)
	subexpNames := re.SubexpNames()
	matchMap := map[string]string{}
	for i, n := range res {
		matchMap[subexpNames[i]] = string(n)
	}

	if strings.ToUpper(matchMap["interfaceType"]) != "USB" {
		return visa, errors.New("visa: interface type was not usb")
	}
	visa.interfaceType = "USB"

	if matchMap["boardIndex"] != "" {
		boardIndex, err := strconv.ParseUint(matchMap["boardIndex"], 0, 16)
		if err != nil {
			return visa, errors.New("visa: boardIndex error")
		}
		visa.boardIndex = int(boardIndex)
	}

	if matchMap["manufacturerID"] != "" {
		manufacturerID, err := strconv.ParseUint(matchMap["manufacturerID"], 0, 16)
		if err != nil {
			return visa, errors.New("visa: manufacturerID error")
		}
		visa.manufacturerID = int(manufacturerID)
	}

	if matchMap["modelCode"] != "" {
		modelCode, err := strconv.ParseUint(matchMap["modelCode"], 0, 16)
		if err != nil {
			return visa, errors.New("visa: modelCode error")
		}
		visa.modelCode = int(modelCode)
	}

	visa.serialNumber = matchMap["serialNumber"]

	if matchMap["interfaceIndex"] != "" {
		interfaceIndex, err := strconv.ParseUint(matchMap["interfaceIndex"], 0, 10)
		if err != nil {
			return visa, errors.New("visa: interfaceIndex error")
		}
		visa.interfaceIndex = int(interfaceIndex)
	}

	if strings.ToUpper(matchMap["resourceClass"]) != "INSTR" {
		return visa, errors.New("visa: resource class was not instr")
	}
	visa.resourceClass = "INSTR"

	return visa, nil

}

// NewDevice searches for device matching vid, pid and serial number. Serial number can be omitted by passing empty string
// If serial number is omitted it will look for first device matching vid & pid
// If a device is detected, it will go over configurations to see if thare is a TMC configuration.
func ListResources() []string {
	var result []string

	// Iterate through available devices. Find all devices that match the given
	// Vendor ID and Product ID.
	ctx := gousb.NewContext()
	defer ctx.Close()

	devs, _ := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		// This anonymous function is called for every device present. Returning
		// true means the device should be opened.
		return true
	})
	for _, d := range devs {
		defer d.Close()
	}

	for _, dev := range devs {

		sn, err := dev.SerialNumber()

		if err != nil {
			continue
		}
		activeCfg, err := dev.ActiveConfigNum()
		if err != nil {
			continue
		}
		cfg, err := dev.Config(activeCfg)
		if err != nil {
			continue
		}
		for _, ifDesc := range cfg.Desc.Interfaces {
			for _, alt := range ifDesc.AltSettings {
				isTmc, _ := usbtmc.CheckTMC(alt)

				if isTmc {
					result = append(result, fmt.Sprintf("USB0::0x%s::0x%s::%s::%d::INSTR", dev.Desc.Vendor, dev.Desc.Product, sn, ifDesc.Number))
				}
			}
		}

	}

	return result
}

func OpenResource(addr string, termchar byte) (*usbtmc.UsbTmc, error) {
	v, err := parseVisaResource(addr)
	if err != nil {
		return nil, err
	}

	dev, err := usbtmc.NewDevice(v.manufacturerID, v.modelCode, v.serialNumber)

	if err != nil {
		return nil, err
	}

	return usbtmc.NewUsbTmc(dev, termchar), nil

}
