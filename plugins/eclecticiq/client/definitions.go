// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package client

import "strings"

type Entity struct {
	InternalID     string `json:"internal_id"`
	StixID         string `json:"stix_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Status         string `json:"status,omitempty"`
	Description    string `json:"description"`
	SimpleOverview string `json:"simple_overview,omitempty"`

	Tags    []string `json:"tags"`
	Sources []string `json:"sources"`

	// STIX-native Timestamps
	StixCreatedAt  string `json:"stix_created_at,omitempty"`  // When the intelligence producer originally created this object
	StixModifiedAt string `json:"stix_modified_at,omitempty"` // When the intelligence producer last modified this object

	EstimatedThreatStartTime string `json:"estimated_threat_start_time"`

	Confidence         string   `json:"confidence,omitempty"`
	Aliases            []string `json:"aliases,omitempty"`
	IntendedEffects    []string `json:"intended_effects,omitempty"`
	LikelyImpact       string   `json:"likely_impact,omitempty"`
	Types              []string `json:"types,omitempty"`
	Product            string   `json:"product,omitempty"`
	Names              []string `json:"names,omitempty"`
	SecurityCompromise string   `json:"security_compromise,omitempty"`
	DiscoveryMethods   []string `json:"discovery_methods,omitempty"`
	Categories         []string `json:"categories,omitempty"`
	CoaType            string   `json:"coa_type,omitempty"`

	CweIds []string `json:"cwe_ids,omitempty"`
	CveIds []string `json:"cve_ids,omitempty"`

	Attack         []*AttackNode    `json:"attack,omitempty"`
	TestMechanisms []*TestMechanism `json:"test_mechanisms,omitempty"`
}

type RelatedEntity struct {
	*Entity
	RelationType string `json:"relation_type"`
	IsOutgoing   bool   `json:"is_outgoing"`
}

type Extract struct {
	Value string `json:"value"`
	Kind  string `json:"kind"`
}

// AttackNode represents a flat, denormalized MITRE ATT&CK association
type AttackNode struct {
	Tactic       string `json:"tactic,omitempty"`
	Technique    string `json:"technique,omitempty"`
	Subtechnique string `json:"subtechnique,omitempty"`
}

type Tactic struct {
	ID         string                `json:"id"`
	Techniques map[string]*Technique `json:"techniques"`
}

type Technique struct {
	TacticID string `json:"tactic_id"`
	ID       string `json:"id"`
}

// TestMechanism maps to STIX 2 'pattern' and 'pattern_type'
type TestMechanism struct {
	Type  string `json:"type"`  // e.g., "stix", "yara", "snort", "sigma", "pcre"
	Value string `json:"value"` // The actual rule or pattern string
}

type Relationship struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Key    string `json:"key"` // e.g., "indicates", "attributed-to"
}

// --- Internal structures for decoding responses safely ---

type rawEntity struct {
	ID   string         `json:"id"`
	Data map[string]any `json:"data"`
	Meta map[string]any `json:"meta"`
}

type rawRelationship struct {
	ID   string `json:"id"`
	Data struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Key    string `json:"key"`
	} `json:"data"`
}

func mapRawEntities(raw []rawEntity) []*Entity {
	var entities []*Entity
	for _, r := range raw {
		entities = append(entities, &Entity{
			InternalID:               r.ID,
			StixID:                   getString(r.Data, "id"),
			Type:                     getString(r.Data, "type"),
			Title:                    getString(r.Data, "title"),
			Status:                   getString(r.Data, "status"),
			Description:              getString(r.Data, "description"),
			SimpleOverview:           getString(r.Data, "short_description"),
			Tags:                     getStringSlice(r.Meta, "tags"),
			Sources:                  []string{getNestedString(r.Data, []string{"information_source", "identity", "name"}, "System")},
			StixCreatedAt:            getString(r.Data, "created_at"), // STIX native created
			StixModifiedAt:           getString(r.Data, "timestamp"),  // STIX native modified (EIQ maps this to data.timestamp)
			EstimatedThreatStartTime: getString(r.Meta, "estimated_threat_start_time"),
			Confidence:               getString(r.Data, "confidence"),
			Aliases:                  extractNamesOrAliases(r.Data, "aliases"),
			IntendedEffects:          getNestedStringSlice(r.Data, []string{"intended_effects"}, "value"),
			LikelyImpact:             getNestedString(r.Data, []string{"likely_impact", "value"}, ""),
			Types:                    extractTypes(r.Data),
			Product:                  getString(r.Data, "product"),
			Names:                    extractNamesOrAliases(r.Data, "names"),
			SecurityCompromise:       getString(r.Data, "security_compromise"),
			DiscoveryMethods:         getNestedStringSlice(r.Data, []string{"discovery_methods"}, "value"),
			Categories:               getNestedStringSlice(r.Data, []string{"categories"}, "value"),
			CoaType:                  getString(r.Data, "coa_type"),
			CweIds:                   getNestedStringSlice(r.Data, []string{"weaknesses"}, "cwe_id"),
			CveIds:                   getNestedStringSlice(r.Data, []string{"vulnerabilities"}, "cve_id"),
			Attack:                   parseAttackData(r.Meta),
			TestMechanisms:           parseTestMechanisms(r.Data),
		})
	}
	return entities
}

func mapRawRelationships(raw []rawRelationship) []*Relationship {
	var rels []*Relationship
	for _, r := range raw {
		rels = append(rels, &Relationship{
			ID:     r.ID,
			Source: r.Data.Source,
			Target: r.Data.Target,
			Key:    r.Data.Key,
		})
	}
	return rels
}

// --- Parsing Helpers ---

func parseAttackData(meta map[string]any) []*AttackNode {
	attacks := getStringSlice(meta, "attacks")
	if len(attacks) == 0 {
		return nil
	}

	// Use a map to deduplicate overlapping nodes
	uniqueNodes := make(map[string]*AttackNode)

	for _, atk := range attacks {
		slashParts := strings.Split(atk, "/")
		attackStr := slashParts[len(slashParts)-1]
		if attackStr == "" {
			continue
		}

		parts := strings.Split(attackStr, ":")
		tacticID := parts[0]

		node := &AttackNode{
			Tactic: tacticID,
		}

		if len(parts) > 1 {
			techID := parts[1] // e.g., "T1098.003" or "T1566"
			if strings.Contains(techID, ".") {
				node.Subtechnique = techID
				node.Technique = strings.Split(techID, ".")[0]
			} else {
				node.Technique = techID
			}
		}

		// Create a unique key for the combination
		key := node.Tactic + "|" + node.Technique + "|" + node.Subtechnique
		uniqueNodes[key] = node
	}

	var res []*AttackNode
	for _, n := range uniqueNodes {
		res = append(res, n)
	}
	return res
}

func parseTestMechanisms(data map[string]any) []*TestMechanism {
	rawTMs, ok := data["test_mechanisms"].([]any)
	if !ok || len(rawTMs) == 0 {
		return nil
	}

	var tms []*TestMechanism
	for _, item := range rawTMs {
		tm, ok := item.(map[string]any)
		if !ok {
			continue
		}

		tmType := getString(tm, "test_mechanism_type")
		switch tmType {
		case "generic":
			// Handles STIX, Sigma, PCRE, Suricata
			genType := getString(tm, "generic_test_mechanism_type")
			val := getNestedString(tm, []string{"specification", "value"}, "")
			if val != "" {
				tms = append(tms, &TestMechanism{Type: genType, Value: val})
			}
		case "snort":
			// Snort rules are inside a "rules" array
			if rulesRaw, ok := tm["rules"].([]any); ok {
				for _, rRaw := range rulesRaw {
					if ruleMap, ok := rRaw.(map[string]any); ok {
						val := getString(ruleMap, "value")
						if val != "" {
							tms = append(tms, &TestMechanism{Type: tmType, Value: val})
						}
					}
				}
			}
		default:
			val := getNestedString(tm, []string{"rule", "value"}, "")
			if val != "" {
				tms = append(tms, &TestMechanism{Type: tmType, Value: val})
			}
		}
	}
	return tms
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getNestedString(m map[string]any, keys []string, fallback string) string {
	curr := m
	for i, k := range keys {
		val, ok := curr[k]
		if !ok {
			return fallback
		}
		if i == len(keys)-1 {
			if str, ok := val.(string); ok {
				return str
			}
			return fallback
		}
		if nestedMap, ok := val.(map[string]any); ok {
			curr = nestedMap
		} else {
			return fallback
		}
	}
	return fallback
}

func getStringSlice(m map[string]any, key string) []string {
	if val, ok := m[key].([]any); ok {
		var slice []string
		for _, v := range val {
			if str, ok := v.(string); ok {
				slice = append(slice, str)
			}
		}
		return slice
	}
	return nil
}

func getNestedStringSlice(m map[string]any, arrayPath []string, targetKey string) []string {
	curr := m
	for _, k := range arrayPath {
		if val, ok := curr[k]; ok {
			if arr, ok := val.([]any); ok {
				var res []string
				for _, item := range arr {
					if obj, ok := item.(map[string]any); ok {
						if target, ok := obj[targetKey].(string); ok {
							res = append(res, target)
						}
					}
				}
				return res
			}
		}
	}
	return nil
}

func extractNamesOrAliases(m map[string]any, key string) []string {
	if val, ok := m[key].([]any); ok {
		var res []string
		for _, item := range val {
			if str, ok := item.(string); ok {
				res = append(res, str)
			} else if obj, ok := item.(map[string]any); ok {
				if nameVal, ok := obj["value"].(string); ok {
					res = append(res, nameVal)
				}
			}
		}
		return res
	}
	return nil
}

func extractTypes(m map[string]any) []string {
	types := getNestedStringSlice(m, []string{"types"}, "value")
	if len(types) == 0 {
		types = getNestedStringSlice(m, []string{"malware_types"}, "value")
	}
	if len(types) == 0 {
		types = getNestedStringSlice(m, []string{"tool_types"}, "value")
	}
	return types
}
