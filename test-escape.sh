#!/bin/bash
# Test script to simulate lsh output

echo -n $'\x1b]9001;HTML_START\x07'
echo "<h1>Test HTML</h1>"
echo -n $'\x1b]9001;HTML_END\x07'
echo ""
