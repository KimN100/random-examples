#!/bin/bash

# Remove any old files
rm -f xx blockfile

# Create some dummy files
# with your name,
# and date
whoami > xx
date >> xx
touch blockfile

# Add the dummy file to the initially empty blockchain.
./blockadd.bash xx blockfile

# Remove the dummy file
rm xx
