#!/usr/bin/env perl

package fastagi;

use strict;
use warnings;
use Asterisk::AGI;
use base 'Net::Server::PreFork';

use constant DEBUG => 0;

my $server = fastagi->new({
	proto => 'tcp',
	port => 4573,
	host => '127.0.0.1',
	min_servers => 5,
	min_spare_servers => 5,
	max_spare_servers => 50,
	max_servers => 200,
	max_requests => 1000,
});

sub process_request {
    my $self = shift;
	STDIN->autoflush( 1 );
	myagi();

}

sub myagi {
	my $status;
	my $file;
	my $agi = new Asterisk::AGI;
	$agi->setcallback(\&mycallback);
	my %input = $agi->ReadParse();

	if (DEBUG) {
		warn "\n==Finished reading AGI vars==\n";
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
	$status = $agi->verbose("Paying back: $file", 0);
	$status = $agi->stream_file($file, '', 0);
	if ($status == -1) {
		warn "Failed to playback file: $file\n";
		goto HANGUP;
	}

HANGUP:
	$agi->hangup();
	return;
}

sub mycallback {
        my ($returncode) = @_;
        warn "User Hungup ($returncode)\n" if (DEBUG);
        return $returncode;
}

$server->run;
