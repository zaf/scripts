#!/usr/bin/env perl
#
# FastAGI Server example
#
# Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 2. See the LICENSE file
# at the top of the source tree.
#

package fastagi;

use strict;
use warnings;
use Asterisk::AGI;
use base 'Net::Server::PreFork';

use constant DEBUG => 0;

fastagi->run(
	{   proto             => 'tcp',
		port              => 4573,
		host              => '127.0.0.1',
		min_servers       => 5,
		min_spare_servers => 5,
		max_spare_servers => 50,
		max_servers       => 500,
		max_requests      => 1000,
		log_level         => 0,
	}
);

sub process_request {
	my $self = shift;
	myagi();
}

sub myagi {
	my $status;
	my $file;
	my $agi   = new Asterisk::AGI;
	my %input = $agi->ReadParse();
	if (DEBUG) {
		warn "\n==AGI environment vars==\n";
		warn "$_: $input{$_}\n" foreach (keys %input);
	}

	if (!exists $input{'arg_1'}) {
		warn "No arguments passed, exiting.\n" if (DEBUG);
		goto HANGUP;
	}
	$file = $input{'arg_1'};

	$status = $agi->channel_status('');
	if ($status == -1) {
		goto HANGUP;
	} elsif ($status != 6) {
		$status = $agi->answer('');
		if ($status == -1) {
			warn "Failed to answer channel\n";
			goto HANGUP;
		}
	}
	$agi->verbose("Paying back: $file", 0);
	$status = $agi->stream_file($file, '', 0);
	if ($status == -1) {
		warn "Failed to playback file: $file\n";
		goto HANGUP;
	}

HANGUP:
	$agi->hangup();
	return;
}
