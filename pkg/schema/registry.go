package schema

import (
	"RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/pkg/model/core"
	"fmt"
)

type EntitySchema struct {
	DgraphType       string                                 // "Host", "Domain", etc.
	DefaultFields    []string                               // Standard-Felder für Queries
	DetailFields     []string                               // Erweiterte Felder (für Get-by-UID etc.)
	UniqueFilters    func(v any) []dgraph.UniqueFieldFilter // Composite Key Builder
	CatalogPredicate core.Predicate                         // Welches Predicate im Katalog
}

var Registry = map[string]EntitySchema{
	"Host": {
		DgraphType: "Host",
		DefaultFields: []string{
			"uid",
			"host.name",
			"host.ip",
			"host.hostname",
			"host.dns_host_name",
			"host.is_domain_controller",
			"host.distinguished_name",
			"host.operating_system",
			"host.operating_system_version",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"host.name",
			"host.ip",
			"host.hostname",
			"host.dns_host_name",
			"host.is_domain_controller",
			"host.distinguished_name",
			"host.operating_system",
			"host.operating_system_version",
			"host.description",
			"host.last_logon_timestamp",
			"host.user_account_control",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateHasHost,
	},
	"Service": {
		DgraphType: "Service",
		DefaultFields: []string{
			"uid",
			"service.name",
			"service.port",
			"service.protocol",
			"service.product",
			"service.version",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"service.name",
			"service.port",
			"service.protocol",
			"service.product",
			"service.version",
			"service.banner",
			"service.state",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateRuns,
	},
	"ActiveDirectory": {
		DgraphType: "ActiveDirectory",
		DefaultFields: []string{
			"uid",
			"active_directory.forest_name",
			"active_directory.forest_functional_level",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"active_directory.forest_name",
			"active_directory.forest_functional_level",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateHasActiveDirectory,
	},
	"Domain": {
		DgraphType: "Domain",
		DefaultFields: []string{
			"uid",
			"domain.name",
			"domain.dns_name",
			"domain.netbios_name",
			"domain.domain_sid",
			"domain.domain_guid",
			"domain.domain_functional_level",
			"domain.forest_functional_level",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"domain.name",
			"domain.dns_name",
			"domain.netbios_name",
			"domain.domain_sid",
			"domain.domain_guid",
			"domain.domain_functional_level",
			"domain.forest_functional_level",
			"domain.fsmo_role_owners",
			"domain.default_containers",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateHasDomain,
	},
	"User": {
		DgraphType: "User",
		DefaultFields: []string{
			"uid",
			"user.name",
			"user.sam_account_name",
			"user.upn",
			"user.sid",
			"user.is_disabled",
			"user.is_locked",
			"user.is_domain_admin",
			"user.risk_score",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"user.name",
			"user.sam_account_name",
			"user.upn",
			"user.sid",
			"user.is_disabled",
			"user.is_locked",
			"user.is_domain_admin",
			"user.is_local_admin",
			"user.kerberoastable",
			"user.asrep_roastable",
			"user.allowed_to_delegate",
			"user.has_spn",
			"user.risk_score",
			"user.risk_reasons",
			"user.last_logon",
			"user.pwd_last_set",
			"user.bad_pwd_count",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateHasUser,
	},
	"Computer": {
		DgraphType: "Computer",
		DefaultFields: []string{
			"uid",
			"computer.name",
			"computer.sid",
			"computer.hostname",
			"computer.is_domain_controller",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"computer.name",
			"computer.sid",
			"computer.hostname",
			"computer.description",
			"computer.is_domain_controller",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateHasHost,
	},
	"Group": {
		DgraphType: "Group",
		DefaultFields: []string{
			"uid",
			"group.name",
			"group.sid",
			"group.group_scope",
			"group.group_type",
			"group.is_privileged",
			"group.can_dcsync",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"group.name",
			"group.sid",
			"group.group_scope",
			"group.group_type",
			"group.is_privileged",
			"group.is_builtin",
			"group.can_dcsync",
			"group.can_rdp",
			"group.can_logon_locally",
			"group.privileges",
			"group.risk_score",
			"group.risk_reasons",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateHasGroup,
	},
	"ServiceAccount": {
		DgraphType: "ServiceAccount",
		DefaultFields: []string{
			"uid",
			"service_account.name",
			"service_account.sid",
			"service_account.sam_account_name",
			"service_account.is_disabled",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		DetailFields: []string{
			"uid",
			"service_account.name",
			"service_account.sid",
			"service_account.sam_account_name",
			"service_account.upn",
			"service_account.is_disabled",
			"service_account.is_locked",
			"created_at",
			"modified_at",
			"dgraph.type",
		},
		CatalogPredicate: core.PredicateHasUser,
	},
}

// --- Helpers ---

func Get(objectType string) (EntitySchema, error) {
	s, ok := Registry[objectType]
	if !ok {
		return EntitySchema{}, fmt.Errorf("unknown entity type: %s", objectType)
	}
	return s, nil
}

func DefaultFields(objectType string) ([]string, error) {
	s, err := Get(objectType)
	if err != nil {
		return nil, err
	}
	return s.DefaultFields, nil
}

func DetailFields(objectType string) ([]string, error) {
	s, err := Get(objectType)
	if err != nil {
		return nil, err
	}
	return s.DetailFields, nil
}

func CatalogPredicate(objectType string) (core.Predicate, error) {
	s, err := Get(objectType)
	if err != nil {
		return "", err
	}
	return s.CatalogPredicate, nil
}
