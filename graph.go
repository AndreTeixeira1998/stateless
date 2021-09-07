package stateless

import (
	"context"
	"fmt"
	"strings"
)

type graph[S State, T Trigger] struct {
}

func (g *graph[S, T]) FormatStateMachine(sm *StateMachine[S, T]) string {
	var sb strings.Builder
	sb.WriteString("digraph {\n\tcompound=true;\n\tnode [shape=Mrecord];\n\trankdir=\"LR\";\n\n")
	for _, sr := range sm.stateConfig {
		if len(sr.Substates) > 0 && sr.Superstate == nil {
			sb.WriteString(g.formatOneCluster(sr))
		} else {
			sb.WriteString(g.formatOneState(sr))
		}
	}
	for _, sr := range sm.stateConfig {
		sb.WriteString(g.formatAllStateTransitions(sm, sr))
	}
	initialState, err := sm.State(context.Background())
	if err == nil {
		sb.WriteString("\n init [label=\"\", shape=point];")
		sb.WriteString(fmt.Sprintf("\n init -> {%v}[style = \"solid\"]", initialState))
	}
	sb.WriteString("\n}")
	return sb.String()
}

func (g *graph[S, T]) formatActions(sr *stateRepresentation[S, T]) string {
	es := make([]string, 0, len(sr.EntryActions)+len(sr.ExitActions)+len(sr.ActivateActions)+len(sr.DeactivateActions))
	for _, act := range sr.ActivateActions {
		es = append(es, fmt.Sprintf("activated / %s", act.Description.String()))
	}
	for _, act := range sr.DeactivateActions {
		es = append(es, fmt.Sprintf("deactivated / %s", act.Description.String()))
	}
	for _, act := range sr.EntryActions {
		if act.Trigger == nil {
			es = append(es, fmt.Sprintf("entry / %s", act.Description.String()))
		}
	}
	for _, act := range sr.ExitActions {
		es = append(es, fmt.Sprintf("exit / %s", act.Description.String()))
	}
	return strings.Join(es, `\n`)
}

func (g *graph[S, T]) formatOneState(sr *stateRepresentation[S, T]) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\t%v [label=\"%v", sr.State, sr.State))
	act := g.formatActions(sr)
	if act != "" {
		sb.WriteString("|")
		sb.WriteString(act)
	}
	sb.WriteString("\"];\n")
	return sb.String()
}

func (g *graph[S, T]) formatOneCluster(sr *stateRepresentation[S, T]) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\nsubgraph cluster_%v {\n\tlabel=\"%v", sr.State, sr.State))
	act := g.formatActions(sr)
	if act != "" {
		sb.WriteString("\n----------\n")
		sb.WriteString(act)
	}
	sb.WriteString("\";\n")
	for _, substate := range sr.Substates {
		sb.WriteString(g.formatOneState(substate))
	}

	sb.WriteString("}\n")
	return sb.String()
}

func (g *graph[S, T]) getEntryActions(ab []actionBehaviour[S, T], t T) []string {
	var actions []string
	for _, ea := range ab {
		if ea.Trigger == nil || *ea.Trigger == t {
			actions = append(actions, ea.Description.String())
		}
	}
	return actions
}

func (g *graph[S, T]) formatAllStateTransitions(sm *StateMachine[S, T], sr *stateRepresentation[S, T]) string {
	var sb strings.Builder
	for _, triggers := range sr.TriggerBehaviours {
		for _, trigger := range triggers {
			switch t := trigger.(type) {
			case *ignoredTriggerBehaviour[T]:
				sb.WriteString(g.formatOneTransition(sr.State, sr.State, t.Trigger, nil, t.Guard))
			case *reentryTriggerBehaviour[S, T]:
				actions := g.getEntryActions(sr.EntryActions, t.Trigger)
				sb.WriteString(g.formatOneTransition(sr.State, t.Destination, t.Trigger, actions, t.Guard))
			case *internalTriggerBehaviour[S, T]:
				actions := g.getEntryActions(sr.EntryActions, t.Trigger)
				sb.WriteString(g.formatOneTransition(sr.State, sr.State, t.Trigger, actions, t.Guard))
			case *transitioningTriggerBehaviour[S, T]:
				var actions []string
				if dest, ok := sm.stateConfig[t.Destination]; ok {
					actions = g.getEntryActions(dest.EntryActions, t.Trigger)
				}
				sb.WriteString(g.formatOneTransition(sr.State, t.Destination, t.Trigger, actions, t.Guard))
			case *dynamicTriggerBehaviour[S, T]:
				// TODO: not supported yet
			}
		}
	}
	return sb.String()
}

func (g *graph[S, T]) formatOneTransition(source S, destination S, trigger T, actions []string, guards transitionGuard) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprint(trigger))
	if len(actions) > 0 {
		sb.WriteString(" / ")
		sb.WriteString(strings.Join(actions, ", "))
	}
	for _, info := range guards.Guards {
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("[%s]", info.Description.String()))
	}
	return g.formatOneLine(fmt.Sprint(source), fmt.Sprint(destination), sb.String())
}

func (g *graph[S, T]) formatOneLine(fromNodeName, toNodeName, label string) string {
	return fmt.Sprintf("\n%s -> %s [style=\"solid\", label=\"%s\"];", fromNodeName, toNodeName, label)
}
