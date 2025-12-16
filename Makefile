.PHONY: build test lint setup-test-dir clean-test-dir

build:
	go build -o fdup .

test:
	go test ./...

lint:
	@echo "Running Go lint..."
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2 run ./...

setup-test-dir:
	@echo "Creating test directory structure..."
	@mkdir -p ./tmp/downloads ./tmp/backup ./tmp/archive
	@# Create test files with duplicate patterns
	@echo "test content 1" > ./tmp/downloads/DSC00001.jpg
	@echo "test content 2" > ./tmp/backup/DSC_00001.jpg
	@echo "test content 3" > ./tmp/downloads/IMG_1234.png
	@echo "test content 4" > ./tmp/backup/IMG-1234.png
	@echo "test content 5" > ./tmp/archive/img1234.png
	@echo "test content 6" > ./tmp/downloads/C0001.mp4
	@echo "test content 7" > ./tmp/backup/C0001_edited.mp4
	@# Initialize fdup (creates .fdup/ with default config and db)
	@cd ./tmp && ../fdup init 2>/dev/null || true
	@# Overwrite config with test patterns
	@printf '%s\n' \
		"patterns:" \
		"  - name: dsc" \
		"    regex: '(DSC[_-]?\\d{5})'" \
		"  - name: img" \
		"    regex: '(IMG[_-]?\\d{4})'" \
		"  - name: video" \
		"    regex: '(C\\d{4})'" \
		"" \
		"ignore:" \
		"  - .git/" \
		"  - .fdup/" \
		"  - '*.tmp'" \
		"" \
		"test:" \
		"  - input: DSC00001.jpg" \
		"    expected: DSC00001" \
		"  - input: IMG_1234.png" \
		"    expected: IMG1234" \
		> ./tmp/.fdup/config.yaml
	@echo "Done! Test directory created at ./tmp"
	@echo "Run: cd ./tmp && ../fdup scan && ../fdup dup"

clean-test-dir:
	@echo "Removing test directory..."
	@rm -rf ./tmp
	@echo "Done!"
