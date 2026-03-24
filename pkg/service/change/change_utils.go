package change

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/redpaths/history"
)

// buildCreatedChange erstellt einen Change-Eintrag für einen neu erstellten Host.
// Alle gesetzten Felder werden als "new_value" mit nil als "old_value" erfasst.
func BuildCreatedChange(host *model.Host, actor string) *history.Change {
	var fields []history.FieldChange

	if host.IP != "" {
		fields = append(fields, history.FieldChange{Field: "ip", OldValue: nil, NewValue: host.IP})
	}
	if host.Hostname != "" {
		fields = append(fields, history.FieldChange{Field: "hostname", OldValue: nil, NewValue: host.Hostname})
	}
	if host.DNSHostName != "" {
		fields = append(fields, history.FieldChange{Field: "dns_host_name", OldValue: nil, NewValue: host.DNSHostName})
	}
	if host.OperatingSystem != "" {
		fields = append(fields, history.FieldChange{Field: "operating_system", OldValue: nil, NewValue: host.OperatingSystem})
	}
	if host.OperatingSystemVersion != "" {
		fields = append(fields, history.FieldChange{Field: "operating_system_version", OldValue: nil, NewValue: host.OperatingSystemVersion})
	}
	if host.DistinguishedName != "" {
		fields = append(fields, history.FieldChange{Field: "distinguished_name", OldValue: nil, NewValue: host.DistinguishedName})
	}
	if host.IsDomainController {
		fields = append(fields, history.FieldChange{Field: "is_domain_controller", OldValue: nil, NewValue: true})
	}

	return &history.Change{
		EntityType:   "Host",
		EntityUID:    host.UID,
		ChangeType:   history.ChangeTypeCreated,
		ChangedBy:    actor,
		ChangeReason: "Host created via upsert",
		Changes:      fields,
	}
}

// buildUpdatedChange erstellt einen Change-Eintrag für einen gemergten Host.
// mergeFields enthält nur die Felder, die tatsächlich überschrieben werden —
// der existing Host liefert die old_values.
func BuildUpdatedChange(
	existing *model.Host,
	mergeFields map[string]interface{},
	actor string,
	reason string,
) *history.Change {
	fieldToOld := map[string]any{
		"host.ip":                       existing.IP,
		"host.dns_host_name":            existing.DNSHostName,
		"host.operating_system":         existing.OperatingSystem,
		"host.operating_system_version": existing.OperatingSystemVersion,
		"host.hostname":                 existing.Hostname,
		"host.distinguished_name":       existing.DistinguishedName,
		"host.is_domain_controller":     existing.IsDomainController,
	}

	// Lesbarer Feldname für die History (ohne "host." prefix)
	displayName := map[string]string{
		"host.ip":                       "ip",
		"host.dns_host_name":            "dns_host_name",
		"host.operating_system":         "operating_system",
		"host.operating_system_version": "operating_system_version",
		"host.hostname":                 "hostname",
		"host.distinguished_name":       "distinguished_name",
		"host.is_domain_controller":     "is_domain_controller",
		"last_seen_at":                  "last_seen_at",
	}

	var fields []history.FieldChange
	for dgraphKey, newVal := range mergeFields {
		name, ok := displayName[dgraphKey]
		if !ok {
			name = dgraphKey
		}
		fields = append(fields, history.FieldChange{
			Field:    name,
			OldValue: fieldToOld[dgraphKey],
			NewValue: newVal,
		})
	}

	return &history.Change{
		EntityType:   "Host",
		EntityUID:    existing.UID,
		ChangeType:   history.ChangeTypeUpdated,
		ChangedBy:    actor,
		ChangeReason: reason,
		Changes:      fields,
	}
}
