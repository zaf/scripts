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

my $debug = 1;

sub echo_test {
	my $self = shift;
	my $agi_input = $self->input();
	my $status;

	if ($debug) {
		warn "Finished reading AGI vars:\n";
		warn "$_:\t\t$agi_input->{$_}\n" foreach (keys %$agi_input);
	}

	if ($agi_input->{'arg_1'} eq '') {
		warn "No arguments passed, exiting.\n" if ($debug);
		goto HANGUP;
	}
	$status = $self->agi->verbose("Staring an echo test.",3);
	if ($status != 0) {
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
	$status = $self->agi->stream_file($agi_input->{'arg_1'}, '', 0);
	if ($status == -1) {
		warn "Failed to playback file $agi_input->{'arg_1'}\n:";
		goto HANGUP;
	}
	$status = $self->agi->exec("echo");
	if ($status != 0) {
		warn "Failed to find application\n";
	}
HANGUP:
	$self->agi->hangup();
}

1;
