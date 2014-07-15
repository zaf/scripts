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
use base 'Net::Server::Fork';

use constant DEBUG => 0;

fastagi->run(
	{   proto       => 'tcp',
		port        => 4573,
		host        => '127.0.0.1',
		max_servers => 2000,
		log_level   => 0,
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
	if ($status != 6) {
		$status = $agi->answer('');
		if ($status == -1) {
			warn "Failed to answer channel\n";
			goto HANGUP;
		}
	}
	$agi->verbose("Playing back: $file", 0);
	$status = $agi->stream_file($file, '1234567890#*', 0);
	warn "Failed to playback file: $file\n" if ($status == -1);

HANGUP:
	$agi->hangup();
	return;
}
