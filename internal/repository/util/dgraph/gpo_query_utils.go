package dgraph

import (
	"RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"RedPaths-server/pkg/model/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dgraph-io/dgo/v210"
)

func GetGPOLinksWithGPOs(
	ctx context.Context,
	tx *dgo.Txn,
	subjectUID string,
	gpoLinkFields []string,
	gpoFields []string,
	queryName string,
) ([]*res.GPOLinkEntry, error) {
	if queryName == "" {
		queryName = "getGPOLinksWithGPOs"
	}

	gpoLinkFieldsStr := strings.Join(gpoLinkFields, "\n\t\t\t\t\t\t")
	gpoFieldsStr := strings.Join(gpoFields, "\n\t\t\t\t\t\t\t\t\t")

	query := fmt.Sprintf(`query %s($subjectUID: string) {
        subject(func: uid($subjectUID)) {
            ~assertion.subject @filter(eq(assertion.predicate, "%s")) {
                gpo_link_assertion_uid: uid
                gpo_link_assertion.predicate: assertion.predicate
                gpo_link_assertion.source: assertion.source
                gpo_link_assertion.confidence: assertion.confidence
                gpo_link_assertion.status: assertion.status
                gpo_link_assertion.timestamp: assertion.timestamp
                gpo_link_assertion.method: assertion.method

                gpolink: assertion.object @filter(type(GPOLink)) {
                    %s
                    ~assertion.subject @filter(eq(assertion.predicate, "%s")) {
                        gpo_assertion_uid: uid
                        gpo_assertion.predicate: assertion.predicate
                        gpo_assertion.source: assertion.source
                        gpo_assertion.confidence: assertion.confidence
                        gpo_assertion.status: assertion.status
                        gpo_assertion.timestamp: assertion.timestamp
                        gpo_assertion.method: assertion.method

                        gpo: assertion.object @filter(type(GPO)) {
                            %s
                        }
                    }
                }
            }
        }
    }`, queryName,
		string(core.PredicateHasGPOLink),
		gpoLinkFieldsStr,
		string(core.PredicateLinksTo),
		gpoFieldsStr,
	)

	log.Printf("[%s] Generated query:\n%s", queryName, query)

	resp, err := tx.QueryWithVars(ctx, query, map[string]string{
		"$subjectUID": subjectUID,
	})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	log.Printf("[%s] Raw response: %s", queryName, string(resp.Json))

	var rawResult map[string]any
	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	subjects := extractArray(rawResult, "subject")
	if len(subjects) == 0 {
		return []*res.GPOLinkEntry{}, nil
	}

	return parseGPOLinkEntries(subjects[0], subjectUID, queryName)
}

func parseGPOLinkEntries(subject map[string]any, subjectUID, queryName string) ([]*res.GPOLinkEntry, error) {
	linkAssertions := extractArray(subject, "~assertion.subject")
	log.Printf("[%s] found %d gpo_link assertions", queryName, len(linkAssertions))

	var entries []*res.GPOLinkEntry

	for _, linkAssertion := range linkAssertions {
		gpoLinks := extractArray(linkAssertion, "gpolink")
		if len(gpoLinks) == 0 {
			log.Printf("[%s] warn: link assertion has no gpolink object", queryName)
			continue
		}

		gpoLinkRaw := gpoLinks[0]

		var gpoLink gpo.Link
		if err := remarshal(gpoLinkRaw, &gpoLink); err != nil {
			return nil, fmt.Errorf("unmarshalling gpo link: %w", err)
		}

		gpoLinkAssertionEntity := buildAssertionFromMap(linkAssertion, "gpo_link_assertion", subjectUID, gpoLink.UID)

		// Now walk the inner assertions to GPO
		innerAssertions := extractArray(gpoLinkRaw, "~assertion.subject")
		log.Printf("[%s] found %d gpo assertions for link %s", queryName, len(innerAssertions), gpoLink.UID)

		for _, gpoAssertion := range innerAssertions {
			gpos := extractArray(gpoAssertion, "gpo")
			if len(gpos) == 0 {
				log.Printf("[%s] warn: gpo assertion has no gpo object", queryName)
				continue
			}

			var linkedGPO gpo.GPO
			if err := remarshal(gpos[0], &linkedGPO); err != nil {
				return nil, fmt.Errorf("unmarshalling gpo: %w", err)
			}

			gpoAssertionEntity := buildAssertionFromMap(gpoAssertion, "gpo_assertion", gpoLink.UID, linkedGPO.UID)

			entries = append(entries, &res.GPOLinkEntry{
				GPOLink:           &gpoLink,
				GPOLinkAssertions: []*core.Assertion{gpoLinkAssertionEntity},
				GPO:               &linkedGPO,
				GPOAssertions:     []*core.Assertion{gpoAssertionEntity},
			})
		}
	}

	return entries, nil
}

// buildAssertionFromMap liest Assertion-Felder mit gegebenem Prefix aus der Map
func buildAssertionFromMap(m map[string]any, prefix, subjectUID, objectUID string) *core.Assertion {
	return &core.Assertion{
		UID:        extractString(m, prefix+"_uid"),
		Predicate:  core.Predicate(extractString(m, prefix+".predicate")),
		Method:     core.Method(extractString(m, prefix+".method")),
		Source:     extractString(m, prefix+".source"),
		Confidence: extractFloat(m, prefix+".confidence"),
		Status:     core.Status(extractString(m, prefix+".status")),
		Timestamp:  extractTime(m, prefix+".timestamp"),
		Subject:    &utils.UIDRef{UID: subjectUID},
		Object:     &utils.UIDRef{UID: objectUID},
	}
}

// remarshal ist ein Helfer: map[string]any → struct via JSON
func remarshal(src any, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
