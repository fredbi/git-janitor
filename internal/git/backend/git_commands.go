package backend

// git_commands.go centralizes all git CLI argument lists used by Runner.run().
//
// Functions are grouped by theme and return []string slices suitable for:
//
//	r.run(ctx, cmdXxx(args)...)

// --- Status & branch info ---

func cmdStatus() []string {
	return []string{"status", "--porcelain=v2", "--branch"}
}

func cmdBranchList() []string {
	return []string{
		"branch", "-a",
		"--format=%(HEAD)|%(refname:short)|%(objectname:short)|%(upstream:short)|%(upstream:track)|%(creatordate:iso-strict)|%(refname)",
	}
}

func cmdBranchMerged(target string) []string {
	return []string{"branch", "--merged", target, "--format=%(refname:short)"}
}

func cmdRemoteBranchMerged(target string) []string {
	return []string{"branch", "-r", "--merged", target, "--format=%(refname:short)"}
}

func cmdCherry(target, branch string) []string {
	return []string{"cherry", target, branch}
}

func cmdSymbolicRef(ref string) []string {
	return []string{"symbolic-ref", "--short", ref}
}

func cmdRevParseVerify(ref string) []string {
	return []string{"rev-parse", "--verify", "--quiet", ref}
}

func cmdRevParseAbbrev(ref string) []string {
	return []string{"rev-parse", "--abbrev-ref", ref}
}

// --- Remote & fetch ---

func cmdRemoteVerbose() []string {
	return []string{"remote", "-v"}
}

func cmdFetchAll() []string {
	return []string{"fetch", "--all"}
}

func cmdFetchAllTags() []string {
	return []string{"fetch", "--all", "--tags", "--force"}
}

func cmdFetchRemote(remote string) []string {
	return []string{"fetch", remote}
}

func cmdFetchRefspec(remote, refspec string) []string {
	return []string{"fetch", remote, refspec}
}

// --- Stash ---

func cmdStashSave(message string) []string {
	if message != "" {
		return []string{"stash", "push", "--include-untracked", "-m", message}
	}

	return []string{"stash", "push", "--include-untracked"}
}

func cmdStashPop() []string {
	return []string{"stash", "pop"}
}

func cmdStashDrop(ref string) []string {
	return []string{"stash", "drop", ref}
}

func cmdStashList() []string {
	// Use --format to get structured output: ref<TAB>ISO-date<TAB>subject
	return []string{"stash", "list", "--format=%gd\t%aI\t%gs"}
}

// --- Log ---

func cmdLastCommitDate() []string {
	return []string{"log", "-1", "--format=%aI"}
}

func cmdLastCommitMessage() []string {
	return []string{"log", "-1", "--format=%s"}
}

// --- Tags ---

func cmdTagList() []string {
	return []string{
		"tag", "-l",
		"--format=%(objecttype)|%(refname:short)|%(objectname:short)|%(*objectname:short)|%(creatordate:iso-strict)|%(contents:subject)|%(if)%(contents:signature)%(then)signed%(else)unsigned%(end)",
	}
}

func cmdLsRemoteTags(remote string) []string {
	return []string{"ls-remote", "--tags", remote}
}

func cmdIsAncestor(commit, branch string) []string {
	return []string{"merge-base", "--is-ancestor", commit, branch}
}

// --- Merge & rebase checks (plumbing) ---

func cmdMergeTree(target, branch string) []string {
	return []string{"merge-tree", "--write-tree", "--name-only", target, branch}
}

func cmdMergeTreeWithBase(parent, current, commit string) []string {
	return []string{"merge-tree", "--write-tree", "--merge-base=" + parent, current, commit}
}

func cmdCommitTree(tree, parent string) []string {
	return []string{"commit-tree", tree, "-p", parent, "-m", "rebase-check"}
}

func cmdMergeBase(a, b string) []string {
	return []string{"merge-base", a, b}
}

func cmdRevListReverse(rangeSpec string) []string {
	return []string{"rev-list", "--reverse", rangeSpec}
}

// --- Actions (mutating) ---

func cmdPullFFOnly() []string {
	return []string{"pull", "--ff-only"}
}

func cmdRebase(target string) []string {
	return []string{"rebase", target}
}

func cmdRebaseAbort() []string {
	return []string{"rebase", "--abort"}
}

func cmdMerge(source string) []string {
	return []string{"merge", source}
}

func cmdMergeAbort() []string {
	return []string{"merge", "--abort"}
}

func cmdPushForceWithLease(remote, branch string) []string {
	return []string{"push", "--force-with-lease", remote, branch}
}

func cmdRemoteSetURL(remoteName, newURL string) []string {
	return []string{"remote", "set-url", remoteName, newURL}
}

func cmdRenameRemote(oldName, newName string) []string {
	return []string{"remote", "rename", oldName, newName}
}

func cmdDeleteBranch(name string) []string {
	return []string{"branch", "-D", name}
}

func cmdDeleteRemoteBranch(remote, branch string) []string {
	return []string{"push", remote, "--delete", branch}
}

func cmdRenameBranch(oldName, newName string) []string {
	return []string{"branch", "-m", oldName, newName}
}

func cmdPushBranchUpstream(remote, branch string) []string {
	return []string{"push", "-u", remote, branch}
}

func cmdPushTag(remote, tag string) []string {
	return []string{"push", remote, tag}
}

func cmdGC() []string {
	return []string{"gc"}
}

func cmdGCAggressive() []string {
	return []string{"gc", "--aggressive"}
}

// --- Details (on-demand) ---

func cmdLogMessage(ref string) []string {
	return []string{"log", "-1", "--format=%s", ref}
}

func cmdDiffStat(rangeSpec string) []string {
	return []string{"diff", "--stat", rangeSpec}
}

func cmdStashShow(ref string) []string {
	return []string{"stash", "show", "--include-untracked", ref}
}

// --- Staging & committing ---

func cmdResetHead() []string {
	return []string{"reset", "HEAD"}
}

func cmdAddAll() []string {
	return []string{"add", "-A"}
}

func cmdCommit(message string) []string {
	return []string{"commit", "-m", message}
}

func cmdCheckout(branch string) []string {
	return []string{"checkout", branch}
}

// --- Worktree ---

func cmdWorktreeList() []string {
	return []string{"worktree", "list", "--porcelain"}
}

func cmdWorktreeAdd(path, branch string) []string {
	return []string{"worktree", "add", path, branch}
}

func cmdWorktreeAddNewBranch(path, newBranch, startPoint string) []string {
	return []string{"worktree", "add", "-b", newBranch, path, startPoint}
}

func cmdWorktreeRemove(path string) []string {
	return []string{"worktree", "remove", "--force", path}
}

// --- Traits ---

func cmdIsShallow() []string {
	return []string{"rev-parse", "--is-shallow-repository"}
}

// --- Health & size ---

func cmdFSCK() []string {
	return []string{"fsck", "--connectivity-only", "--no-progress", "--no-dangling"}
}

func cmdCountObjects() []string {
	return []string{"count-objects", "-v"}
}

func cmdConfigGet(key string) []string {
	return []string{"config", "--get", key}
}

func cmdConfigGetRegexp(pattern string) []string {
	return []string{"config", "--show-scope", "--get-regexp", pattern}
}

func cmdRevParseGitDir() []string {
	return []string{"rev-parse", "--git-dir"}
}

func cmdRevListDiskUsage() []string {
	return []string{"rev-list", "--disk-usage", "--all"}
}

// --- File stats ---

func cmdLsTree() []string {
	return []string{"ls-tree", "-r", "-l", "HEAD"}
}

func cmdRevListObjects() []string {
	return []string{"rev-list", "--objects", "--all"}
}

func cmdCatFileBatchCheck() []string {
	return []string{"cat-file", "--batch-check=%(objecttype) %(objectsize) %(objectname)"}
}

func cmdDiffNumstat(emptyTree string) []string {
	return []string{"diff", "--numstat", emptyTree, "HEAD"}
}

// --- Activity ---

func cmdRevListCount(since string) []string {
	return []string{"rev-list", "--count", "--since=" + since, "HEAD"}
}

func cmdShortlog(since string) []string {
	return []string{"shortlog", "-sne", "--since=" + since, "HEAD"}
}

func cmdHashObject() []string {
	return []string{"hash-object", "-t", "tree", "/dev/null"}
}
