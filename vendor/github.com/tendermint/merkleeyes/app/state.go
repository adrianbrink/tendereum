package app

import "github.com/tendermint/tmlibs/merkle"

// State represents the app states, separating the commited state (for queries)
// from the working state (for CheckTx and AppendTx)
type State struct {
	committed  merkle.Tree
	deliverTx  merkle.Tree
	checkTx    merkle.Tree
	persistent bool
}

func NewState(tree merkle.Tree, persistent bool) State {
	return State{
		committed:  tree,
		deliverTx:  tree.Copy(),
		checkTx:    tree.Copy(),
		persistent: persistent,
	}
}

func (s State) Committed() merkle.Tree {
	return s.committed
}

func (s State) Append() merkle.Tree {
	return s.deliverTx
}

func (s State) Check() merkle.Tree {
	return s.checkTx
}

// Hash updates the tree
func (s *State) Hash() []byte {
	return s.deliverTx.Hash()
}

// Commit save persistent nodes to the database and re-copies the trees
func (s *State) Commit() []byte {
	var hash []byte
	if s.persistent {
		hash = s.deliverTx.Save()
	} else {
		hash = s.deliverTx.Hash()
	}

	s.committed = s.deliverTx
	s.deliverTx = s.committed.Copy()
	s.checkTx = s.committed.Copy()
	return hash
}
