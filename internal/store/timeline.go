// Timeline assembly: merges node, edge, resolved-curiosity, and tag
// events into one chronologically ordered stream per iteration.
package store

import (
	"github.com/camggould/speechflow/internal/core"
)

// Timeline returns a chronologically ordered sequence of TimelineEvents
// for an iteration: node creations, edge creations, curiosity resolutions,
// root touches, and tag additions. Tags carry the creation timestamp of
// their host node since the schema does not store per-tag timestamps.
func (s *Store) Timeline(iterationID string) ([]core.TimelineEvent, error) {
	nodes, err := s.ListNodes(iterationID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(iterationID)
	if err != nil {
		return nil, err
	}

	events := make([]core.TimelineEvent, 0, len(nodes)+len(edges))
	for _, n := range nodes {
		id := n.ID
		kind := core.TimelineNodeAdded
		payload := map[string]any{
			"kind":  string(n.Kind),
			"title": n.Title,
		}
		if n.Kind == core.NodeKindRootRef {
			kind = core.TimelineRootTouched
			if n.RootID != nil {
				payload["root_id"] = *n.RootID
			}
		}
		events = append(events, core.TimelineEvent{
			Ts:      n.CreatedAt,
			Kind:    kind,
			NodeID:  &id,
			Payload: payload,
		})
		for _, tag := range n.Tags {
			tagPayload := map[string]any{"tag": tag}
			events = append(events, core.TimelineEvent{
				Ts:      n.CreatedAt,
				Kind:    core.TimelineTagAdded,
				NodeID:  &id,
				Payload: tagPayload,
			})
		}
		if n.ResolvedByNodeID != nil {
			payload := map[string]any{"resolved_by": *n.ResolvedByNodeID}
			// We don't store the moment of resolution separately; use the
			// resolver node's creation time as the best available proxy.
			ts := n.CreatedAt
			for _, m := range nodes {
				if m.ID == *n.ResolvedByNodeID {
					ts = m.CreatedAt
					break
				}
			}
			cid := n.ID
			events = append(events, core.TimelineEvent{
				Ts:      ts,
				Kind:    core.TimelineCuriosityResolved,
				NodeID:  &cid,
				Payload: payload,
			})
		}
	}
	for _, e := range edges {
		id := e.ID
		events = append(events, core.TimelineEvent{
			Ts:     e.CreatedAt,
			Kind:   core.TimelineEdgeAdded,
			EdgeID: &id,
			Payload: map[string]any{
				"from": e.FromNode,
				"to":   e.ToNode,
				"kind": string(e.Kind),
			},
		})
	}

	// Stable chronological sort.
	for i := 1; i < len(events); i++ {
		for j := i; j > 0 && events[j-1].Ts.After(events[j].Ts); j-- {
			events[j-1], events[j] = events[j], events[j-1]
		}
	}
	return events, nil
}
