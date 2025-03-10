package ofctrl

import (
	"antrea.io/libOpenflow/openflow13"
	"antrea.io/libOpenflow/util"
)

type MeterFlag int
type MeterType uint16

const (
	MeterKbps  MeterFlag = 0b0001
	MeterPktps MeterFlag = 0b0010
	MeterBurst MeterFlag = 0b0100
	MeterStats MeterFlag = 0b1000

	MeterDrop         MeterType = 1      /* Drop packet. */
	MeterDSCPRemark   MeterType = 2      /* Remark DSCP in the IP header. */
	MeterExperimenter MeterType = 0xFFFF /* Experimenter meter band. */
)

type MeterBundleMessage struct {
	message *openflow13.MeterMod
}

func (m *MeterBundleMessage) resetXid(xid uint32) util.Message {
	m.message.Xid = xid
	return m.message
}

type Meter struct {
	Switch      *OFSwitch
	ID          uint32
	Flags       MeterFlag
	MeterBands  []*util.Message
	isInstalled bool
}

func (self *Meter) Type() string {
	return "meter"
}

func (self *Meter) GetFlowInstr() openflow13.Instruction {
	meterInstr := openflow13.NewInstrMeter(self.ID)
	return meterInstr
}

func (self *Meter) AddMeterBand(meterBands ...*util.Message) {
	if self.MeterBands == nil {
		self.MeterBands = make([]*util.Message, 0)
	}
	self.MeterBands = append(self.MeterBands, meterBands...)
	if self.isInstalled {
		self.Install()
	}
}

func (self *Meter) Install() error {
	command := openflow13.OFPMC_ADD
	if self.isInstalled {
		command = openflow13.OFPMC_MODIFY
	}
	meterMod := self.getMeterModMessage(command)

	if err := self.Switch.Send(meterMod); err != nil {
		return err
	}

	// Mark it as installed
	self.isInstalled = true

	return nil
}

func (self *Meter) getMeterModMessage(command int) *openflow13.MeterMod {
	meterMod := openflow13.NewMeterMod()
	meterMod.MeterId = self.ID
	meterMod.Flags = uint16(self.Flags)

	for _, mb := range self.MeterBands {
		// Add the meterBands to meter
		meterMod.AddMeterBand(*mb)
	}
	meterMod.Command = uint16(command)

	return meterMod
}

func (self *Meter) GetBundleMessage(command int) *MeterBundleMessage {
	meterMod := self.getMeterModMessage(command)
	return &MeterBundleMessage{meterMod}
}

func (self *Meter) Delete() error {
	if self.isInstalled {
		meterMod := openflow13.NewMeterMod()
		meterMod.MeterId = self.ID
		meterMod.Command = openflow13.OFPMC_DELETE
		if err := self.Switch.Send(meterMod); err != nil {
			return err
		}
		// Mark it as unInstalled
		self.isInstalled = false
	}

	// Delete meter from switch cache
	return self.Switch.DeleteMeter(self.ID)
}

func newMeter(id uint32, flags MeterFlag, ofSwitch *OFSwitch) *Meter {
	return &Meter{
		ID:     id,
		Flags:  flags,
		Switch: ofSwitch,
	}
}
