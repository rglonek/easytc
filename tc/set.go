package tc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func Set(r *Rule) error {
	// list qdisc
	rules, err := ListRules()
	if err != nil {
		return err
	}

	// check if, and initialize qdisc if needed
	isInit := false
	for _, q := range rules.Qdisc {
		if q.Kind == nil || *q.Kind != "prio" {
			continue
		}
		if q.Options == nil || q.Options.PrioBands == nil {
			continue
		}
		isInit = true
		break
	}
	if !isInit {
		for _, iface := range rules.Interfaces {
			if iface == "lo" {
				continue
			}
			out, err := exec.Command("tc", "qdisc", "add", "dev", iface, "root", "handle", "1:", "prio", "bands", "16").CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %s", err, string(out))
			}
		}
	}

	// find existing rule if one already there
	// create/replace qdisc rule
	newFlowId := 3
	for _, rule := range rules.Rules {
		if rule.FlowID != nil {
			fid := strings.Split(*rule.FlowID, ":")
			if len(fid) != 2 {
				continue
			}
			if a, _ := strconv.Atoi(fid[1]); a >= newFlowId {
				newFlowId = a + 1
			}
		}
		if rule.Iface == nil || *rule.Iface != *r.Iface {
			continue
		}
		if (rule.LatencyMs == nil && r.LatencyMs != nil) || (rule.LatencyMs != nil && r.LatencyMs == nil) {
			continue
		}
		if r.LatencyMs != nil && *r.LatencyMs != *rule.LatencyMs {
			continue
		}
		if (rule.PacketLossPct == nil && r.PacketLossPct != nil) || (rule.PacketLossPct != nil && r.PacketLossPct == nil) {
			continue
		}
		if r.PacketLossPct != nil {
			pctTr, _ := strconv.ParseFloat(*r.PacketLossPct, 64)
			if fmt.Sprintf("%0.2f", pctTr) != *rule.PacketLossPct {
				continue
			}
		}
		if (rule.LinkSpeedRateBytes == nil && r.LinkSpeedRateBytes != nil) || (rule.LinkSpeedRateBytes != nil && r.LinkSpeedRateBytes == nil) {
			continue
		}
		if r.LinkSpeedRateBytes != nil && *r.LinkSpeedRateBytes != *rule.LinkSpeedRateBytes {
			continue
		}
		r.QdiscHandle = rule.QdiscHandle
		r.QdiscNo = rule.QdiscNo
		r.FlowID = rule.FlowID
		break
	}
	if r.FlowID == nil {
		r.FlowID = StringToPtr("1:" + strconv.Itoa(newFlowId))
		r.QdiscHandle = StringToPtr(strconv.Itoa(newFlowId) + "0:")
	}
	params := []string{"qdisc", "replace", "dev", *r.Iface, "parent", *r.FlowID, "handle", *r.QdiscHandle, "netem"}
	if r.LatencyMs != nil {
		params = append(params, "delay", *r.LatencyMs+"ms")
	}
	if r.LinkSpeedRateBytes != nil {
		params = append(params, "rate", *r.LinkSpeedRateBytes+"bps")
	}
	if r.PacketLossPct != nil {
		params = append(params, "loss", *r.PacketLossPct+"%")
	}
	out, err := exec.Command("tc", params...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(out))
	}

	// add filter rule as defined, if one does not exist; if one does exist, remove/replace with new flowID
	filterFound := false
	for _, rule := range rules.Rules {
		if r.Iface == nil || rule.Iface == nil || *r.Iface != *rule.Iface {
			continue
		}
		if (r.SourceIP != nil && rule.SourceIP == nil) || (r.SourceIP == nil && rule.SourceIP != nil) {
			continue
		}
		if (r.DestinationIP != nil && rule.DestinationIP == nil) || (r.DestinationIP == nil && rule.DestinationIP != nil) {
			continue
		}
		if (r.SourcePort != nil && rule.SourcePort == nil) || (r.SourcePort == nil && rule.SourcePort != nil) {
			continue
		}
		if (r.DestinationPort != nil && rule.DestinationPort == nil) || (r.DestinationPort == nil && rule.DestinationPort != nil) {
			continue
		}
		if r.SourceIP != nil && *r.SourceIP != *rule.SourceIP {
			continue
		}
		if r.DestinationIP != nil && *r.DestinationIP != *rule.DestinationIP {
			continue
		}
		if r.SourcePort != nil && *r.SourcePort != *rule.SourcePort {
			continue
		}
		if r.DestinationPort != nil && *r.DestinationPort != *rule.DestinationPort {
			continue
		}
		// we are here, the rule had been found, change flowid to match r.FlowID
		out, err := exec.Command("tc", "filter", "replace", "dev", *rule.Iface, "protocol", "ip", "parent", "1:0", "prio", "3", "handle", *rule.FilterHandle, "u32", "flowid", *r.FlowID).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, string(out))
		}
		filterFound = true
		break
	}

	if !filterFound {
		// we are here, the filter rule is not found, create a new filter for r.FlowID
		params := []string{"filter", "add", "dev", *r.Iface, "protocol", "ip", "parent", "1:0", "prio", "3", "u32"}
		if r.SourceIP != nil {
			params = append(params, "match", "ip", "src", *r.SourceIP)
		}
		if r.DestinationIP != nil {
			params = append(params, "match", "ip", "dst", *r.DestinationIP)
		}
		if r.SourcePort != nil {
			params = append(params, "match", "ip", "sport", *r.SourcePort, "0xffff")
		}
		if r.DestinationPort != nil {
			params = append(params, "match", "ip", "dport", *r.DestinationPort, "0xffff")
		}
		params = append(params, "flowid", *r.FlowID)
		out, err := exec.Command("tc", params...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", err, string(out))
		}
	}

	CleanupUnusedQdisc()
	return nil
}
