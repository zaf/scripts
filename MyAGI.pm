package MyAGI;

#
# Copyright (C) 2014, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 2. See the LICENSE file
# at the top of the source tree.
#

use strict;
use warnings;

use base 'Asterisk::FastAGI';

my $debug = 0;

sub myagi {
	my $self      = shift;
	my $agi_input = $self->input();
	my $status;

	if ($debug) {
		warn "Finished reading AGI vars:\n";
		warn "$_:\t\t$agi_input->{$_}\n" foreach (keys %$agi_input);
	}

	if ($self->param('file') eq '') {
		warn "No arguments passed, exiting.\n" if ($debug);
		goto HANGUP;
	}

	$status = $self->agi->channel_status('');
	if ($status == -1) {
		goto HANGUP;
	} elsif ($status != 6) {
		$status = $self->agi->answer('');
		if ($status == -1) {
			warn "Failed to answer channel\n";
			goto HANGUP;
		}
	}
	$status = $self->agi->verbose("Paying back: " . $self->param('file'), 0);
# 	if ($status != 1) {
# 		goto HANGUP;
# 	}
	$status = $self->agi->stream_file($self->param('file'), '', 0);
	if ($status == -1) {
		warn "Failed to playback file: " . $self->param('file') . "\n";
		goto HANGUP;
	}

HANGUP:
	$self->agi->hangup();
	return;
}

1;
