package sekurlsa

// AVL-tree traversal helper.
//
// Vista+ Kerberos and TSPkg replaced the legacy doubly-linked
// LIST_ENTRY walker with an NT-runtime AVL tree backed by
// `RTL_AVL_TABLE` allocations. The non-obvious layout pypykatz
// documents:
//
//	Each AVL node has an RTL_BALANCED_LINKS at offset 0x00:
//	  +0x00 Parent      pointer
//	  +0x08 LeftChild   pointer
//	  +0x10 RightChild  pointer
//	  +0x18 Balance     int8 (signed) + 3 reserved
//
//	IMMEDIATELY AFTER the RTL_BALANCED_LINKS, at +0x20, the node
//	carries the user payload. Microsoft's `RtlInsertElement-
//	GenericTableAvl` allocates `sizeof(RTL_BALANCED_LINKS) + UserSize`
//	per node; the first user-payload bytes start at +0x20. For
//	Kerberos the user payload is `KIWI_KERBEROS_LOGON_SESSION *`
//	(8 bytes), so the actual session struct lives at the address
//	stored at node+0x20 — NOT at the node itself.
//
// The tree root is the `RTL_AVL_TABLE.BalancedRoot.RightChild`
// (offset 0x10 of the table). The table itself is a sentinel
// whose Parent is self-referential and Left/Right anchor the
// real tree.
//
// References:
//   pypykatz: pypykatz/lsadecryptor/package_commons.py walk_avl
//   Microsoft: nt!RtlEnumerateGenericTableAvl in ntoskrnl
//
// The pre-Vista Kerberos walker (flat linked list) is gone — every
// modern target is Vista+ so we ship only the AVL path.

const (
	// rtlBalancedLinksLeftOffset is the byte offset to the LeftChild
	// pointer inside an RTL_BALANCED_LINKS struct.
	rtlBalancedLinksLeftOffset uint64 = 0x08
	// rtlBalancedLinksRightOffset is the byte offset to the
	// RightChild pointer inside an RTL_BALANCED_LINKS struct.
	rtlBalancedLinksRightOffset uint64 = 0x10
	// avlNodeUserDataOffset is the offset of the user-payload bytes
	// past the RTL_BALANCED_LINKS prefix. For Kerberos this is the
	// `KIWI_KERBEROS_LOGON_SESSION *` pointer the walker dereferences
	// to reach the actual session struct.
	avlNodeUserDataOffset uint64 = 0x20
)

// walkAVL traverses the AVL tree rooted at root and invokes visit
// on every node in in-order traversal. Returns no error — visit
// failures are silently absorbed because a corrupted dump's tree
// shape can be partial.
//
// maxNodes bounds the traversal to defeat malformed dumps with
// loops in the tree (which a real RTL_AVL_TABLE never has, but
// we don't fully trust dump bytes). The visited-set defends
// against the same shape of corruption.
//
// The starting `root` argument is the actual tree-root pointer —
// callers convert from `RTL_AVL_TABLE.BalancedRoot.RightChild` by
// reading `*(table_va + 0x10)` before calling walkAVL.
func walkAVL(r *reader, root uint64, maxNodes int, visit func(node uint64)) {
	if root == 0 || maxNodes <= 0 {
		return
	}

	visited := make(map[uint64]struct{}, 64)
	var walk func(addr uint64)
	walk = func(addr uint64) {
		if addr == 0 || len(visited) >= maxNodes {
			return
		}
		if _, ok := visited[addr]; ok {
			return
		}
		visited[addr] = struct{}{}

		// In-order: left, self, right. Pypykatz uses pre-order; the
		// order doesn't matter for credential enumeration.
		if left, err := readPointer(r, addr+rtlBalancedLinksLeftOffset); err == nil {
			walk(left)
		}
		visit(addr)
		if right, err := readPointer(r, addr+rtlBalancedLinksRightOffset); err == nil {
			walk(right)
		}
	}
	walk(root)
}

// readAVLTreeRoot dereferences a pointer to the first tree node
// rooted under an `RTL_AVL_TABLE` whose address is `tableVA`. The
// `BalancedRoot` sentinel sits at offset 0; the actual tree root
// is its `RightChild` (offset 0x10).
//
// Returns 0 when the read fails or the tree is empty.
func readAVLTreeRoot(r *reader, tableVA uint64) uint64 {
	root, err := readPointer(r, tableVA+rtlBalancedLinksRightOffset)
	if err != nil {
		return 0
	}
	return root
}
