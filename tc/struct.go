package tc

import (
	"log"
)

/*
=-=-=-=-= filter:lo =-=-=-=-=
[]
=-=-=-=-= filter:enp0s5 =-=-=-=-=
[
  {
    "parent": "1:",
    "protocol": "ip",
    "pref": 3,
    "kind": "u32",
    "chain": 0,
    "options": null
  },
  {
    "parent": "1:",
    "protocol": "ip",
    "pref": 3,
    "kind": "u32",
    "chain": 0,
    "options": {
      "fh": "800:",
      "ht_divisor": 1,
      "order": null,
      "key_ht": null,
      "bkt": null,
      "flowid": null,
      "not_in_hw": null,
      "match": null
    }
  },
  {
    "parent": "1:",
    "protocol": "ip",
    "pref": 3,
    "kind": "u32",
    "chain": 0,
    "options": {
      "fh": "800::800",
      "ht_divisor": null,
      "order": 2048,
      "key_ht": "800",
      "bkt": "0",
      "flowid": "1:3",
      "not_in_hw": true,
      "match": [
        {
          "value": "c0a80001",
          "mask": "ffffffff",
          "offmask": "",
          "off": 12
        },
        {
          "value": "c0a80032",
          "mask": "ffffffff",
          "offmask": "",
          "off": 16
        },
        {
          "value": "bb80bb8",
          "mask": "ffffffff",
          "offmask": "",
          "off": 20
        }
      ]
    }
  }
]
=-=-=-=-= qdisc =-=-=-=-=
[
  {
    "kind": "noqueue",
    "handle": "0:",
    "dev": "lo",
    "root": true,
    "refcnt": 2,
    "parent": null,
    "options": {
      "bands": null,
      "priomap": null,
      "multiqueue": null,
      "limit": null,
      "delay": null,
      "loss-random": null,
      "rate": null,
      "ecn": null,
      "gap": null
    }
  },
  {
    "kind": "prio",
    "handle": "1:",
    "dev": "enp0s5",
    "root": true,
    "refcnt": 2,
    "parent": null,
    "options": {
      "bands": 3,
      "priomap": [
        1,
        2,
        2,
        2,
        1,
        2,
        0,
        0,
        1,
        1,
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "multiqueue": false,
      "limit": null,
      "delay": null,
      "loss-random": null,
      "rate": null,
      "ecn": null,
      "gap": null
    }
  },
  {
    "kind": "netem",
    "handle": "30:",
    "dev": "enp0s5",
    "root": null,
    "refcnt": null,
    "parent": "1:3",
    "options": {
      "bands": null,
      "priomap": null,
      "multiqueue": null,
      "limit": 1000,
      "delay": {
        "delay": 0.1,
        "jitter": 0,
        "correlation": 0
      },
      "loss-random": {
        "loss": 0.1,
        "correlation": 0
      },
      "rate": {
        "rate": 100000,
        "packetoverhead": 0,
        "cellsize": 0,
        "celloverhead": 0
      },
      "ecn": false,
      "gap": 0
    }
  }
]
*/

type Qdisc struct {
	Kind    *string       `json:"kind"`
	Handle  *string       `json:"handle"`
	Dev     *string       `json:"dev"`
	Root    *bool         `json:"root"`
	Refcnt  *int          `json:"refcnt"`
	Parent  *string       `json:"parent"`
	Options *QdiscOptions `json:"options"`
}

type QdiscOptions struct {
	PrioBands       *int             `json:"bands"`
	PrioMap         *[]int           `json:"priomap"`
	PrioMultiqueue  *bool            `json:"multiqueue"`
	NetemLimit      *int             `json:"limit"`
	NetemDelay      *NetemDelay      `json:"delay"`
	NetemLossRandom *NetemLossRandom `json:"loss-random"`
	NetemRate       *NetemRate       `json:"rate"`
	NetemCorrupt    *NetemCorrupt    `json:"corrupt"`
	NetemEcn        *bool            `json:"ecn"`
	NetemGap        *float64         `json:"gap"`
}

type NetemCorrupt struct {
	Corrupt     float64 `json:"corrupt"`
	Correlation float64 `json:"correlation"`
}

type NetemDelay struct {
	Delay       float64 `json:"delay"`
	Jitter      float64 `json:"jitter"`
	Correlation float64 `json:"correlation"`
}

type NetemLossRandom struct {
	Loss        float64 `json:"loss"`
	Correlation float64 `json:"correlation"`
}

type NetemRate struct {
	Rate           int `json:"rate"`
	PacketOverhead int `json:"packetoverhead"`
	CellSize       int `json:"cellsize"`
	CellOverhead   int `json:"celloverhead"`
}

type Filter struct {
	Iface    string         `json:"interface"`
	Parent   *string        `json:"parent"`
	Protocol *string        `json:"protocol"`
	Pref     *int           `json:"pref"`
	Kind     *string        `json:"kind"`
	Chain    *int           `json:"chain"`
	Options  *FilterOptions `json:"options"`
}

type FilterOptions struct {
	FH          *string       `json:"fh"`
	HtDivisor   *int          `json:"ht_divisor"`
	Order       *int          `json:"order"`
	KeyHt       *string       `json:"key_ht"`
	Bkt         *string       `json:"bkt"`
	FlowId      *string       `json:"flowid"`
	NotInHw     *bool         `json:"not_in_hw"`
	Match       FilterMatches `json:"match"`
	MatchParsed struct {
		SourceIPMask *string `json:"source_ip_mask"`
		DestIPMask   *string `json:"dest_ip_mask"`
		SourcePort   *int    `json:"source_port"`
		DestPort     *int    `json:"dest_port"`
	} `json:"match_parsed"`
}

type FilterMatch struct {
	Value   string `json:"value"`
	Mask    string `json:"mask"`
	Offmask string `json:"offmask"`
	Offset  int    `json:"off"`
}

type FilterMatches []*FilterMatch

type Rules struct {
	Interfaces []string
	Qdisc      []*Qdisc
	Filters    []*Filter
	Rules      []*Rule
}

type Rule struct {
	// set, delete
	Iface           *string
	SourceIP        *string
	SourcePort      *string
	DestinationIP   *string
	DestinationPort *string
	// set only
	LatencyMs          *string
	PacketLossPct      *string
	LinkSpeedRateBytes *string
	CorruptPct         *string
	// output only parameters
	FlowID       *string
	FilterNo     int
	FilterHandle *string
	QdiscNo      int
	QdiscHandle  *string
}

func logf(verbose bool, format string, v ...interface{}) {
	if !verbose {
		return
	}
	log.Printf("VERBOSE: "+format, v...)
}
