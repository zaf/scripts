#!/usr/bin/env perl
#
# FastAGI Server example using POE
#
# Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 2. See the LICENSE file
# at the top of the source tree.
#

package fastagi;

use warnings;
use strict;
use URI;
use Asterisk::AGI;
use POE qw(Component::Server::TCP);

use constant DEBUG => 0;

POE::Component::Server::TCP->new(
	Address => '127.0.0.1',
	Port    => 4573,
	Acceptor => \&myagi,
);

sub myagi {
	my $socket = $_[ARG0];
	*STDIN  = \*{ $socket };
	*STDOUT = \*{ $socket };
	STDIN->autoflush(1);
	STDOUT->autoflush(1);
	my $status;
	my $file;
	my $agi   = new Asterisk::AGI;
	my %input = $agi->ReadParse();
	if (DEBUG) {
		warn "\n==AGI environment vars==\n";
		warn "$_: $input{$_}\n" foreach (keys %input);
	}
	my $uri = URI->new($input{"request"});
	my %query = $uri->query_form;
	if (! exists $query{"file"}) {
		warn "No file name parameter passed, exiting.\n" if (DEBUG);
		goto HANGUP;
	}
	$file = $query{"file"};

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

POE::Kernel->run;
exit;
