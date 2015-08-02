#!/usr/bin/env perl
#
# FastAGI Server with TLS support example
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
use URI;
use Asterisk::AGI;
use base 'Net::Server::PreFork';

use constant DEBUG => 0;

fastagi->run(
	{   proto             => 'ssl',
		SSL_key_file      => "secret.key",
		SSL_cert_file     => "public.crt",
		port              => 4574,
		host              => '127.0.0.1',
		min_servers       => 10,
		min_spare_servers => 10,
		max_spare_servers => 100,
		max_servers       => 2000,
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
	my $agi   = Asterisk::AGI->new();
	my %input = $agi->ReadParse();
	if (DEBUG) {
		warn "\n==AGI environment vars==\n";
		warn "$_: $input{$_}\n" foreach (keys %input);
	}
	my $uri   = URI->new($input{"request"});
	my %query = $uri->query_form;
	if (!exists $query{"file"}) {
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
