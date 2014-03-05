#!/usr/bin/env perl

# Calculate average sessions and duration in FastAGI benchmark results
#
# Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 2. See the LICENSE file
# at the top of the source tree.

use strict;
use warnings;

my $runs = 0;
my $active  = 0;
my $duration = 0;

while (<STDIN>) {
	next if /^#/;
	/^\d+,(\d+),(\d+)$/;
	$active+=$1;
	$duration+=$2;
	$runs++;
}

if (!$runs) {
	print "No data found.\n";
	exit;
} else {
	print "Average values after $runs runs:\n";
	print "Active Sessions:  " . int($active/$runs) . "\n";
	print "Session Duration: " . $duration/$runs . " ns\n";
}

