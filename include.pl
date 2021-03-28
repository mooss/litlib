#!/usr/bin/env perl

# This poorly put together script parses a file (supposedly a org-mode file),
# extracts the dependencies defined in it, attempts to match them with the arguments
# given to it ($cpp and $noweb) and prints out the resulting C++ code (or the first
# error that occured).

# There are certainly a lot of unknown caveats.
# One I do know for sure is that =noweb-ref= in property drawers are not supported.

# For an example about how this is supposed to be used, look at the include code
# block in interprete.org, along with its various invocations.

###########
# Imports #
###########
use strict;
use warnings;
use v5.14; # for say.
use constant {
    MAX_DEPTH => 2
};
use Text::ParseWords qw/quotewords/;

###################
# Early functions #
###################
sub stop {
    my $msg = shift;
    say '#error "' . $msg . '"';
    exit();
}

sub comment {
    say '// ' . shift;
}

sub extract_parameters {
    my $parameters_string = shift;
    $parameters_string =~ s/^\s*://
        or stop "Parameters string `$parameters_string` does not start with `:`";
    my @parameters = quotewords ':', 1, $parameters_string;
    my %parameters = map {s/\s+$//; my ($h, @t) = quotewords '\s+', 0, $_; $h => \@t}
        @parameters;
    return \%parameters;
}

sub any {
    my ($predicate, $list) = @_;
    foreach(@$list){
        if($predicate->($_)){
            return 1;
        }
    }
    return 0;
}

sub none {
    return ! any @_;
}

######################
# Arguments handling #
######################
stop("Usage: $0 filenames flags")
    if @ARGV != 2;

stop 'You must provide flags using the noweb syntax.'
    if $ARGV[1] eq '' or ! $ARGV[1] =~ '^:';

my $filenames = $ARGV[0];
my %flags = %{extract_parameters $ARGV[1]};

# Check for mandatory inclusion flags.
my @inclusion_flags = qw/cpp noweb/;
stop('At least one inclusion flag is required. Inclusion flags are '
     . join(', ', map {':' . $_} @inclusion_flags) . '.')
    if none(sub{defined $flags{$_[0]} and @{$flags{$_[0]}} > 0}, \@inclusion_flags);

# Extract individual flags.
my $cpp = $flags{cpp} || [];
my $noweb = $flags{noweb} || [];
my $c_string = $flags{'c-string'};

stop(':c-string is incompatible with :cpp, it should only be used with :noweb.')
    if defined $c_string and @$cpp > 0;

# Apparently, it is considered "redefining" if I define a sub in an if and in its else, hence the closure.
my $debug = sub {};
if(defined $flags{debug}) {
    $debug = sub {
        comment 'DBG: ' . shift;
    }
}

##############################
# File reading and "parsing" #
##############################
sub merge_into_left {
    my ($left, $right) = @_;
    $left = {} if !defined $left;
    while(my ($arg, $values) = each(%$right)) {
        push @{$left->{$arg}}, @$values;
    }
}

my (@global_lines, %global_named_blocks, %global_dependencies, %reffed);
sub lines_and_blocks {
    foreach my $filename (split / /, $filenames) {
        open(my $file, '<', $filename)
            or stop("Bad filename ($filename).");
        while(<$file>) {
            push @global_lines, $_;
        }
    }
    my %named;
    for(my $num = 0; $num < @global_lines; ++$num) {
        my $line = $global_lines[$num];
        if($line =~ /^\s*#\+name: (.*)$/) {
            my $name = $1; chomp($name);
            if($global_lines[$num + 1] =~ /^\s*#\+begin_src/) {
                stop "Duplicated named code block `$name`."
                    if $name ne 'include' # `include` is special because it's used as a shortcut for this script.
                    and exists $global_named_blocks{$name};
                $named{$num + 1} = $name;
                push @{$global_named_blocks{$name}}, $num + 2;
            }

        } elsif($line =~ /^\s*#\+begin_src .+ (:.*)/) {
            $debug->("Code block start -> $line");
            my $args = extract_parameters($1);
            my $name = $args->{'noweb-ref'}[0];
            if(defined $name) {
                push @{$global_named_blocks{$name}}, $num + 1;
                if(exists $named{$num}) {
                    push @{$reffed{$name}}, $named{$num};
                    # So this thing is both reffed and named ? I have no idea why.
                }
            }

        } elsif($line =~ /^\s*#\+depends:([^\s]+)\s+(.*)/) {
            stop "Dependency duplicate `$1`." if exists $global_dependencies{$1};
            $global_dependencies{$1} = extract_parameters($2);
        }
    }

    # Inflate dependencies for reffed code blocks.
    while(my ($noweb_ref, $components) = each(%reffed)) {
        $global_dependencies{$noweb_ref} = {} if !exists $global_dependencies{$noweb_ref};
        foreach(@$components) {
            merge_into_left($global_dependencies{$noweb_ref}, $global_dependencies{$_} // {});
        }
    }
}
lines_and_blocks();

###########################
# Dependencies resolution #
###########################
my @cpp_;
my @noweb_;
my %seen_noweb;
my %seen_cpp;
sub extract_dependencies {
    foreach my $name (@_) {
        if(!$seen_noweb{$name}++) { # Kinda weird trick to use a hashtable as a set.
            my $deps = $global_dependencies{$name};
            # I'm not sure why this script used to stop when no dependencies were declared.
            # stop("No dependencies declared for `$name`.") if !defined $deps;
            foreach(@{$deps->{cpp}}) {
                push @cpp_, $_ if !$seen_cpp{$_}++;
            }

            # Code blocks must be included *after* their dependencies, but to avoid double inclusions,
            # the named code blocks who are subsets of reffed codeblocks must be marked as seen.
            if(defined $reffed{$name}) {
                foreach(@{$reffed{$name}}) { $seen_noweb{$_}++; }
            }
            extract_dependencies(@{$deps->{noweb}}) if defined $deps->{noweb};
            push @noweb_, $name;
        }
    }
}

extract_dependencies(@$noweb);
foreach(@$cpp) {
    push @cpp_, $_ if !$seen_cpp{$_}++;
}

########################
# Code blocks printing #
########################
# The output difference between C strings and standard noweb inclusions is handled here.
my $print_cb;
if(defined $c_string) {
    $print_cb = sub {
        my ($prefix, $line) = @_;
        print '"';
        # Not sure if prefix makes any sense in this context.
        print $prefix;
        chomp $line;
        print $line;
        print '\n';
        say '"';
    }
} else {
    $print_cb = sub {
        my ($prefix, $line) = @_;
        print $prefix;
        print $line;
    }
}

sub print_codeblock_rec {
    my ($name, $prefix, $depth) = @_;
    stop "Cannot find code block named `$name`, I only know of [" .
        join(', ', map {'`' . $_ . '`'} keys %global_named_blocks) . '].'
        unless exists $global_named_blocks{$name};
    stop "Maximum noweb inclusion depth reached (" . MAX_DEPTH
        . "). Check for recursive noweb inclusions or increase MAX_DEPTH."
        unless $depth <= MAX_DEPTH;

    for my $linum (@{$global_named_blocks{$name}}) {
        until ((my $line = $global_lines[$linum]) =~ /^\s*#\+end_src/ || $linum >= @global_lines) {
            if ($line =~ /(\s*)<<(.+)>>/) { # Assuming that noweb is always enabled.
                print_codeblock_rec($2, $prefix . $1, $depth + 1);
            } else {
                # Lines starting with `#+`or `*` are a syntax specific to org mode.
                # To handle code blocks with lines starting with those syntaxes, org-mode automatically
                # escapes them by adding a comma in front.
                # Hence the next line to undo this escaping and print the code blocks as they are
                # supposed to be.
                $line =~ s/^,(,*(?:#\+|\*))/$1/; # Only the first comma is removed, the other must be kept.
                $print_cb->($prefix, $line);
            }
            ++$linum;
        }
    }
}

# Not printing twice if also in wanted noweb-ref (included codeblocks can be printed twice).
my %already_printed;
sub print_codeblock {
    my $name = shift;
    return if exists $already_printed{$name};
    $already_printed{$name} = undef;
    if(defined $reffed{$name}) {
        foreach(@{$reffed{$name}}) { $already_printed{$_} = undef; }
    }
    print_codeblock_rec($name, '', 0);
}

foreach(@cpp_) {
    say "#include <$_>";
}
foreach(@noweb_) {
    print_codeblock($_);
}
