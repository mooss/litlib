#!/usr/bin/env perl
# Extract filters from pandoc.org.
# Should be called by make, not manually.

use strict;
use warnings;
use v5.14;
use File::Basename;
use constant {
    ORG => 0,
    LUA_CODE => 1,
};

die 'This script takes no arguments.' unless @ARGV == 0;

my $source = 'pandoc.org';
open(my $panhandle, '<encoding(UTF-8)', $source)
    or die "Failed to open `$source`.";

sub make_necessary_dir {
    my $dirname = dirname(shift);
    mkdir $dirname unless -d $dirname;
}

my $current = ORG;
my $dest_handle;
my $dest_filename;

foreach my $line (<$panhandle>) {
    if($current == ORG) {
        if($line =~ /^#\+begin_src lua.* :tangle\s+([^\s]+)/) {
            $dest_filename = $1;
            make_necessary_dir($dest_filename);
            open($dest_handle, '>:encoding(UTF-8)', $dest_filename)
                or die "Failed to open `$1`.";
            $current = LUA_CODE;
        }
    } elsif($current == LUA_CODE) {
        if($line =~ /^#\+end_src/) {
            close $dest_handle;
            say "Wrote `$dest_filename`.";
            $dest_filename = '';
            $current = ORG;
        } else {
            print $dest_handle $line;
        }
    } else {
        die "\$current value invalid: `$current`.";
    }
}
