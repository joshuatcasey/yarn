.PHONY: assemble prepare-commit prune

assemble:
	@./assemble/assemble.sh $(id) $(artifactPath) $(buildpackTomlPath)

prepare-commit:
	@./prepare-commit/prepare-commit.sh $(metadataPath) $(tarballName) $(id) $(version) $(os)

prune:
	@./prune/prune.sh
