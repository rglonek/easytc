package main

import (
	"easytc/tc"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/bestmethod/inslice"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/jessevdk/go-flags"
	"golang.org/x/term"
)

type command struct {
	Set   cmdSet   `command:"set" description:"create a tc rule"`
	Del   cmdDel   `command:"del" description:"delete a tc rule"`
	Reset cmdReset `command:"reset" description:"remove all tc rules"`
	Show  struct {
		Iface cmdShowIface `command:"iface" description:"list interfaces"`
		Rules cmdShowRules `command:"rules" description:"list rules"`
		All   cmdShowAll   `command:"all" description:"list all interfaces, rules, qdisc and filters in json format"`
	} `command:"show" description:"list tc rules or interfaces"`
}

type cmdSet struct {
	Interface          *string `short:"i" long:"interface" description:"specify an interface for the rule"`
	SourceIP           *string `short:"s" long:"src-ip" description:"optional: filter by source IP"`
	DestinationIP      *string `short:"d" long:"dst-ip" description:"optional: filter by destination IP"`
	SourcePort         *string `short:"S" long:"src-port" description:"optional: filter by source port"`
	DestinationPort    *string `short:"D" long:"dst-port" description:"optional: filter by destination port"`
	LatencyMs          *string `short:"l" long:"latency-ms" description:"optional: specify latency (number) of milliseconds"`
	PacketLossPct      *string `short:"p" long:"loss-pct" description:"optional: specify packet loss percentage"`
	LinkSpeedRateBytes *string `short:"e" long:"rate-bytes" description:"optional: specify link speed rate, in bytes"`
	Verbose            bool    `long:"verbose" description:"enable verbose logging"`
}

type cmdDel struct {
	Interface       *string `short:"i" long:"interface" description:"specify an interface for the rule"`
	SourceIP        *string `short:"s" long:"src-ip" description:"filter source IP"`
	DestinationIP   *string `short:"d" long:"dst-ip" description:"filter destination IP"`
	SourcePort      *string `short:"S" long:"src-port" description:"filter source port"`
	DestinationPort *string `short:"D" long:"dst-port" description:"filter destination port"`
	Verbose         bool    `long:"verbose" description:"enable verbose logging"`
}

type cmdReset struct {
	Interface *string `short:"i" long:"interface" description:"optional: specify an interface; default action: all interfaces"`
	Verbose   bool    `long:"verbose" description:"enable verbose logging"`
}

type cmdShowIface struct{}

type cmdShowRules struct {
	Verbose bool `long:"verbose" description:"enable verbose logging"`
}

type cmdShowAll struct {
	Verbose bool `long:"verbose" description:"enable verbose logging"`
}

func main() {
	cmd := &command{}
	_, err := flags.Parse(cmd)
	if err != nil {
		switch err.(type) {
		case *flags.Error:
			fmt.Fprintln(os.Stderr, "Use '--help' for more details.")
		}
		os.Exit(1)
	}
}

func (c *cmdSet) Execute(tail []string) error {
	mods, err := tc.ListKernelMods(c.Verbose)
	if err != nil {
		return err
	}
	if !inslice.HasString(mods, "sch_netem") {
		err = tc.InsertKernelMod(c.Verbose)
		if err != nil {
			return errNoNetem
		}
	}
	return tc.Set(&tc.Rule{
		Iface:              c.Interface,
		SourceIP:           c.SourceIP,
		DestinationIP:      c.DestinationIP,
		SourcePort:         c.SourcePort,
		DestinationPort:    c.DestinationPort,
		LatencyMs:          c.LatencyMs,
		PacketLossPct:      c.PacketLossPct,
		LinkSpeedRateBytes: c.LinkSpeedRateBytes,
	}, c.Verbose)
}

func (c *cmdDel) Execute(tail []string) error {
	mods, err := tc.ListKernelMods(c.Verbose)
	if err != nil {
		return err
	}
	if !inslice.HasString(mods, "sch_netem") {
		err = tc.InsertKernelMod(c.Verbose)
		if err != nil {
			return errNoNetem
		}
	}
	return tc.Delete(&tc.Rule{
		Iface:           c.Interface,
		SourceIP:        c.SourceIP,
		DestinationIP:   c.DestinationIP,
		SourcePort:      c.SourcePort,
		DestinationPort: c.DestinationPort,
	}, c.Verbose)
}

var errNoNetem = errors.New("kernel module 'sch_netem' not found; centos install via `yum install kernel-modules-extra iproute-tc`; reboot may be required")

func (c *cmdReset) Execute(tail []string) error {
	mods, err := tc.ListKernelMods(c.Verbose)
	if err != nil {
		return err
	}
	if !inslice.HasString(mods, "sch_netem") {
		err = tc.InsertKernelMod(c.Verbose)
		if err != nil {
			return errNoNetem
		}
	}
	return tc.Reset(c.Interface, c.Verbose)
}

func (c *cmdShowIface) Execute(tail []string) error {
	data, err := tc.ListIface()
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range data {
		fmt.Println(d)
	}
	return nil
}

func (c *cmdShowRules) Execute(tail []string) error {
	rules, err := tc.ListRules(c.Verbose)
	if err != nil {
		return err
	}
	t := table.NewWriter()
	type renderer func() string
	var render renderer = t.Render
	t.SortBy([]table.SortBy{
		{
			Name: "Iface",
			Mode: table.Asc,
		},
		{
			Name: "FilterHandle",
			Mode: table.Asc,
		},
		{
			Name: "QdiscHandle",
			Mode: table.Asc,
		},
	})
	t.SetStyle(table.StyleDefault)
	tstyle := t.Style()
	tstyle.Options.DrawBorder = false
	tstyle.Options.SeparateColumns = false
	tstyle.Format.Header = text.FormatDefault
	tstyle.Format.Footer = text.FormatDefault
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 1 {
		fmt.Fprintf(os.Stderr, "Couldn't get terminal width (int:%v): %v", width, err)
	} else {
		if width < 40 {
			width = 40
		}
		t.SetAllowedRowLength(width)
	}
	t.AppendHeader(table.Row{"Iface", "SrcIP", "DstIP", "SrcPort", "DstPort", "LatencyMs", "PacketLossPct", "RateBytes", "TcFlowID", "TcQdiscHandle", "TcFilterHandle"})
	for _, rule := range rules.Rules {
		vv := table.Row{
			tc.PtrToString(rule.Iface),
			tc.PtrToString(rule.SourceIP),
			tc.PtrToString(rule.DestinationIP),
			tc.PtrToString(rule.SourcePort),
			tc.PtrToString(rule.DestinationPort),
			tc.PtrToString(rule.LatencyMs),
			tc.PtrToString(rule.PacketLossPct),
			tc.PtrToString(rule.LinkSpeedRateBytes),
			tc.PtrToString(rule.FlowID),
			tc.PtrToString(rule.QdiscHandle),
			tc.PtrToString(rule.FilterHandle),
		}
		t.AppendRow(vv)
	}
	fmt.Println(render())
	fmt.Println()
	return nil
}

func (c *cmdShowAll) Execute(tail []string) error {
	data, err := tc.ListRules(c.Verbose)
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
