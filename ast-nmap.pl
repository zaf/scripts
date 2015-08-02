#!/usr/bin/env perl
#
# AGI Script that scans an IP address using nmap and reports back to the user.
#
# Copyright (C) 2012, Lefteris Zafiris <zaf.000@gmail.com>
#
# This program is free software, distributed under the terms of
# the GNU General Public License Version 2. See the LICENSE file
# at the top of the source tree.
#
# -----
# Usage
# -----
# agi(ast-nmap,[ipaddr]): This will invoke the nmap port scanner, scan the defined IP
# address and report back to the user possible open ports. If ipaddr is not defined the
# script will promt the user to enter an IP.
#

use strict;
use warnings;
use Nmap::Parser;
$| = 1;

# ----------------------------- #
#   User defined parameters:    #
# ----------------------------- #
# nmap options                  #
my $nmap_args = "";

# TTS application               #
my $tts_app = "flite";

# Verbose debugging messages    #
my $debug = 0;

# ----------------------------- #

my %AGI;
my $name;
my $ipaddr;
my @result;
my $timeout = 5000;
my $nmap    = `/usr/bin/which nmap`;
my $range   = qr/
	^([01]?\d\d?|2[0-4]\d|25[0-5])\.([01]?\d\d?|2[0-4]\d|25[0-5])\.
	([01]?\d\d?|2[0-4]\d|25[0-5])\.([01]?\d\d?|2[0-4]\d|25[0-5])$
/x;

# Store AGI input #
while (<STDIN>) {
	chomp;
	last if (!length);
	$AGI{$1} = $2 if (/^agi_(\w+)\:\s+(.*)$/);
}
($AGI{arg_1}) = @ARGV;
$name = " -- $AGI{request}:";

die "$name nmap is missing. Aborting.\n" if (!$nmap);
chomp($nmap);

# Answer channel if not already answered #
print "CHANNEL STATUS\n";
@result = checkresponse();
if ($result[0] == 4) {
	print "ANSWER\n";
	checkresponse();
}

if (!length($AGI{arg_1})) {
	# Promt user to enter IP. #
	speak("Enter the IP address you wish to scan. When done press the pound key");
	while (length($ipaddr) < 15) {
		print "WAIT FOR DIGIT $timeout\n";
		@result = checkresponse();
		my $digit  = chr($result[0]);
		$ipaddr   .= $digit if ($digit =~ /\d/);
		$ipaddr   .= "."    if ($digit eq "*");
		last                if ($digit eq "#" || $result[0] <= 0);
	}
} else {
	$ipaddr = $AGI{arg_1};
}

if ($ipaddr !~ /$range/) {
	speak("Invalid Address: $ipaddr");
	die "$name Invalid Address: $ipaddr";
}
speak("Please hold. Scanning.");
warn "$name Scanning $ipaddr\n" if ($debug);

my $np = Nmap::Parser->new();
$np->callback(\&host_handler);
$np->parsescan($nmap, $nmap_args, $ipaddr);
speak("Scan complete. Thank you.");
exit;

sub host_handler {
	my $host = shift;
	speak("Host " . $host->addr() . " is " . $host->status());
	return if ($host->status() ne "up");
	speak("Found " . $host->tcp_port_count() . " ports open");
	speak($host->tcp_service($_)->name . " on port $_") foreach $host->tcp_open_ports();
}

sub speak {
	my $text = shift;
	print "EXEC $tts_app \"$text\"\n";
	my @res = checkresponse();
	warn "$name failed to find TTS app.\n" if ($res[0] == -1);
}

sub checkresponse {
	my $input = <STDIN>;
	my @values;

	chomp $input;
	if ($input =~ /^200/) {
		$input =~ /result=(-?\d+)\s?(.*)$/;
		if (!length($1)) {
			warn "$name Command failed: $input\n";
			@values = (-1, -1);
		} else {
			warn "$name Command returned: $input\n" if ($debug);
			@values = ("$1", "$2");
		}
	} else {
		warn "$name Unexpected result: $input\n";
		@values = (-1, -1);
	}
	return @values;
}
