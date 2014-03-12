#!/usr/bin/env perl

# FastAGI server example in Perl
#
# Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 2. See the LICENSE file
# at the top of the source tree.
#

use strict;
use warnings;

use MyAGI;

print "Starting FastAGI server...\n";

MyAGI->run(
	host              => '0.0.0.0',
	port              => '4573',
	log_level         => '0',
	min_servers       => '10',
	min_spare_servers => '10',
	max_spare_servers => '50',
	max_servers       => '1000',
);
