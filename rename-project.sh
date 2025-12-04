#!/bin/bash

# rename-project.sh - replace import paths in the project
# Reads current module name from go.mod and replaces it with new one (from argument)

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Check argument
if [ $# -eq 0 ]; then
    echo "Usage: $0 <new-module-name>"
    echo ""
    echo "Example:"
    echo "  $0 github.com/mycompany/my-gateway"
    echo "  $0 gitlab.com/team/api-gateway"
    echo ""
    echo "This will:"
    echo "  1. Read current module name from go.mod"
    echo "  2. Replace all imports with new module name"
    echo "  3. Update go.mod"
    exit 1
fi

NEW_MODULE="$1"

# Validate new module name
if [[ ! "$NEW_MODULE" =~ ^[a-zA-Z0-9._/-]+$ ]]; then
    error "Invalid module name. Use format: domain.com/user/project"
fi

# Check if go.mod exists
[ ! -f "go.mod" ] && error "go.mod not found. Run this script from project root."

# Read current module name from go.mod
OLD_MODULE=$(head -1 go.mod | awk '{print $2}')
[ -z "$OLD_MODULE" ] && error "Cannot parse module name from go.mod"

# Check if it's already the same module
if [ "$NEW_MODULE" = "$OLD_MODULE" ]; then
    success "Module name already set to: $NEW_MODULE"
    echo "Nothing to do."
    exit 0
fi

echo "================================================"
echo "  Rename Project Imports"
echo "================================================"
echo ""
info "Old module: $OLD_MODULE"
info "New module: $NEW_MODULE"
echo ""

# Count changes
FILES_CHANGED=0
TOTAL_REPLACEMENTS=0

# Function to replace in file
replace_in_file() {
    local file="$1"
    local count=$(grep -c "$OLD_MODULE" "$file" 2>/dev/null || true)

    if [ "$count" -gt 0 ]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|$OLD_MODULE|$NEW_MODULE|g" "$file"
        else
            sed -i "s|$OLD_MODULE|$NEW_MODULE|g" "$file"
        fi
        echo "  - $file ($count replacements)"
        FILES_CHANGED=$((FILES_CHANGED + 1))
        TOTAL_REPLACEMENTS=$((TOTAL_REPLACEMENTS + count))
    fi
}

# 1. Update go.mod
info "Updating go.mod..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "1s|^module .*|module $NEW_MODULE|" go.mod
else
    sed -i "1s|^module .*|module $NEW_MODULE|" go.mod
fi
success "go.mod updated"
echo ""

# 2. Update all Go files
info "Updating Go files..."
while IFS= read -r -d '' file; do
    replace_in_file "$file"
done < <(find . -type f -name "*.go" \
    -not -path "./vendor/*" \
    -not -path "./.git/*" \
    -not -path "./bin/*" \
    -print0)
success "Go files updated"
echo ""

# 3. Update README if it contains old module
info "Checking README.md..."
if [ -f "README.md" ]; then
    replace_in_file "README.md"
fi
echo ""

# 4. Update documentation
info "Checking documentation..."
if [ -d "docs" ]; then
    while IFS= read -r -d '' file; do
        replace_in_file "$file"
    done < <(find docs -type f -name "*.md" -print0)
fi
echo ""

# 5. go mod tidy
info "Running go mod tidy..."
if go mod tidy; then
    success "go mod tidy completed"
else
    error "go mod tidy failed"
fi

echo ""
echo "================================================"
echo "  Summary"
echo "================================================"
echo ""
echo "Files changed:        $FILES_CHANGED"
echo "Total replacements:   $TOTAL_REPLACEMENTS"
echo ""

if [ "$FILES_CHANGED" -eq 0 ]; then
    success "No changes needed - imports already use correct module name"
else
    success "Project imports renamed successfully!"
fi

echo ""
