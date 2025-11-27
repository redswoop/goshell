#!/bin/bash
# Comprehensive test driver for goshell

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "goshell Test Suite"
echo "========================================="
echo ""

# Test 1: Regular output works
echo -e "${YELLOW}Test 1: Regular terminal output${NC}"
echo "Type: echo 'Hello World'"
echo "Expected: See 'Hello World' in terminal"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: Regular output${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 2: HTML command shows link in terminal
echo -e "${YELLOW}Test 2: HTML command shows link${NC}"
echo "Type: lsh"
echo "Expected: See blue clickable 'View HTML Output #N' link in terminal"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: HTML link not visible${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 3: HTML pane appears with content
echo -e "${YELLOW}Test 3: HTML pane shows content${NC}"
echo "Expected: HTML panel above terminal shows file listing with sort buttons"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: HTML pane not showing${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 4: Refresh page - links restored
echo -e "${YELLOW}Test 4: Refresh page${NC}"
echo "Action: Refresh the browser page (Cmd+R or F5)"
echo "Expected: See the same 'View HTML Output #N' link in terminal history"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: Links not restored after refresh${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 5: Clicking link loads HTML
echo -e "${YELLOW}Test 5: Click link to load HTML${NC}"
echo "Action: Click the 'View HTML Output #N' link"
echo "Expected: HTML panel appears/updates with file listing"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: Clicking link doesn't load HTML${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 6: Interactive HTML buttons work
echo -e "${YELLOW}Test 6: Interactive HTML buttons${NC}"
echo "Action: Click one of the sort buttons (Name, Date, Size, Reverse)"
echo "Expected: New HTML output appears with re-sorted listing"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: HTML buttons don't work${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 7: Full-screen VT mode (vim)
echo -e "${YELLOW}Test 7: Full-screen VT applications${NC}"
echo "Type: vim (then :q to quit)"
echo "Expected: vim opens normally, exits cleanly"
echo "After exit: buffer should NOT show vim escape sequences"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: VT mode broken${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 8: Buffer clean after vim
echo -e "${YELLOW}Test 8: Buffer clean after full-screen app${NC}"
echo "Action: Refresh the page"
echo "Expected: No vim escape sequences or junk in replay buffer"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: Buffer contains VT junk${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 9: HTML works after refresh
echo -e "${YELLOW}Test 9: HTML after refresh${NC}"
echo "Type: lsh"
echo "Action: Observe new link appears"
echo "Action: Refresh page"
echo "Action: Click the NEW link"
echo "Expected: HTML panel loads correctly"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: HTML broken after refresh${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 10: Multiple HTML outputs
echo -e "${YELLOW}Test 10: Multiple HTML outputs${NC}"
echo "Type: lsh (run it 3 times)"
echo "Expected: Three different 'View HTML Output #N' links with increasing numbers"
echo "Action: Click each link"
echo "Expected: Each link shows its corresponding HTML content"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: Multiple HTML outputs broken${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

# Test 11: No escape sequences in HTML content
echo -e "${YELLOW}Test 11: Clean HTML content${NC}"
echo "Action: Right-click on HTML panel, Inspect Element"
echo "Expected: HTML should be clean CSS/HTML, no VT100 escape sequences like \\x1b"
echo "Press ENTER when verified, or 'f' to fail"
read -r response
if [ "$response" = "f" ]; then
    echo -e "${RED}FAILED: HTML contains escape sequences${NC}"
    exit 1
fi
echo -e "${GREEN}PASSED${NC}"
echo ""

echo "========================================="
echo -e "${GREEN}ALL TESTS PASSED!${NC}"
echo "========================================="
