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

sub echo_test {
	my $self = shift;
	my $agi_input = $self->input();

	warn "Finished reading AGI vars:\n";
	warn "$_: $agi_input->{$_}\n" foreach (keys %$agi_input);

	if ($agi_input->{'arg_1'} eq '') {
		warn "No arguments passed, exiting.\n";
		goto HANGUP;
	}
	$self->agi->verbose("Staring an echo test.",3);
	if ($self->agi->channel_status() != 6) {
		$self->agi->answer();
	}
	$self->agi->stream_file($agi_input->{'arg_1'});
	$self->agi->exec("echo");
HANGUP:
	$self->agi->hangup();
}

1;
