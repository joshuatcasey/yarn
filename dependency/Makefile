.PHONY: compile test retrieve metadata

compile:
	# @./compilation/compile.sh $(version) $(tarball_name) $(os)
	@./compilation/compile.sh $(version) $(output_dir) $(target)

test:
	@./test/test.sh $(tarball_name) $(version)

retrieve:
	@./retrieval/retrieve.sh $(buildpackTomlPath) $(output)

metadata:
	@./metadata/metadata.sh $(version) $(id) $(name) $(output)
