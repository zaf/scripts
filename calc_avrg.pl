#!/usr/bin/env perl

# Calculate average number of sessions and duration in FastAGI benchmark results
#
# Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 2. See the LICENSE file
# at the top of the source tree.

use strict;
use warnings;
use autodie;

if (!$ARGV[0] or $ARGV[0] eq '-h' or $ARGV[0] eq '--help') {
	print "Calculate average values from FastAGI benchmark log files\nUsage: $0 [FILES]\n";
	exit;
}

my @file_list = @ARGV;

foreach my $file (@file_list) {
	my $runs = 0;
	my $active  = 0;
	my $duration = 0;
	open(my $csvfile, "<", "$file");
	while (<$csvfile>) {
		if (/^\d+,(\d+),(\d+)$/) {
			$active+=$1;
			$duration+=$2;
			$runs++;
		}
	}
	print "\nResults for $file:\n";
	if (!$runs) {
		print "No data found.\n";
		next;
	} else {
		print "Average values after $runs runs:\n";
		print "Active Sessions:  " . int($active/$runs) . "\n";
		print "Session Duration: " . $duration/$runs . " ns\n";
	}
	close $csvfile;
}
