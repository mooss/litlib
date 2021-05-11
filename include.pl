#!/usr/bin/env perl

# This poorly put together script parses a file (supposedly a org-mode file),
# extracts the dependencies defined in it, attempts to match them with the arguments
# given to it (:cpp and :noweb) and prints out the resulting C++ code (or the first
# error that occured).

# An "include as C string" feature (:c-string) and a tangle (:tangle) feature have
# been hastily retrofitted in this script, making it look more and more like the
# creature of Frankenstein.

# There are certainly a lot of unknown caveats.
# One I do know for sure is that =noweb-ref= in property drawers are not supported.

# For an example about how this is supposed to be used, look at README.org.

###########
# Imports #
###########
use strict;
use warnings;
use v5.14; # for say.
use constant {
    MAX_DEPTH => 3
};
use Text::ParseWords qw/quotewords/;
use File::Basename;

###################
# Early functions #
###################
my $exit_code = 0;
# Proper arguments parsing is bypassed for this particular argument because the error
# is susceptible to be used before the arguments are parsed.
if($ARGV[1] =~ /:exit-with-error/) {
    $exit_code = 23;
}

sub stop {
    my $msg = shift;
    say '#error "' . $msg . '"';
    exit($exit_code);
}

sub comment {
    say '// ' . shift;
}

sub extract_parameters {
    my $parameters_string = shift;
    $parameters_string =~ s/^\s*://
        or stop "Parameters string `$parameters_string` does not start with `:`";
    # (^| ) allows : to be used inside values or parameters.
    my @parameters = quotewords '(^| ):', 1, $parameters_string;
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

# Check for mandatory flags.
my @inclusion_flags = qw/cpp noweb/;
my @standalone_boolean_flags = qw/tangle/;
my @mandatory_flags = @inclusion_flags; push @mandatory_flags, @standalone_boolean_flags;
# Bad error message but it would be too much work to make it clearer.
stop('At least one of the following flags is required: '
     . join(', ', map {':' . $_} @mandatory_flags) . '.')
    if none(sub{defined $flags{$_[0]} and @{$flags{$_[0]}} > 0}, \@inclusion_flags)
    and none(sub{defined $flags{$_[0]}}, \@standalone_boolean_flags);

# The inclusion of a tangling operation in this script is dubious since that's not something an "include"
# script should do but it was easy enough to put in place and include.pl is supposed to be temporary
# though I expect it will take a long time to replace it with a cleaner approach.

# Extract individual flags.
my $cpp = $flags{cpp} || [];
my $noweb = $flags{noweb} || [];
my $c_string = $flags{'c-string'};
my $tangle = defined $flags{tangle};

# Add additional filenames.
if(defined $flags{defs}) {
    # Deduplication should be put in place some day.
    $filenames .= ' ' . join(' ', @{$flags{defs}});
}

stop(':c-string is incompatible with :cpp, it should only be used with :noweb.')
    if defined $c_string and @$cpp > 0;

stop(':tangle is incompatible with :cpp and :noweb.')
    if $tangle and (@$cpp > 0 or @$noweb > 0);

# Apparently, it is considered "redefining" if I define a sub in an if and in its else, hence the closure.
my $debug = sub {};
if(defined $flags{debug}) {
    $debug = sub {
        comment 'DBG: ' . shift;
    }
}

#####################
# Utility functions #
#####################
sub make_necessary_dir {
    my $dirname = dirname(shift);
    mkdir $dirname unless -d $dirname;
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

my (@global_lines, %global_named_blocks, %global_dependencies, %reffed, %global_tangle);
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

        } elsif($line =~ /^\s*#\+begin_src [^:]+ (:.*)/) {
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

        } elsif($tangle and $line =~ /^\s*#\+tangle:([^\s]+)\s+(.*)/) {
            # Tangling has its own #+tangle: directive because org-mode has no way to resolve the #+depends:
            # syntax.
            # Thus it's better to bypass entirely the :tangle noweb argument.
            $global_tangle{$1} = $2;
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
my @cpp_dependencies;
my @noweb_dependencies;
my %seen_noweb;
my %seen_cpp;
sub extract_dependencies {
    foreach my $name (@_) {
        if(!$seen_noweb{$name}++) { # Kinda weird trick to use a hashtable as a set.
            if(!defined $global_named_blocks{$name}) {
                # This is a spontaneous dependency, a dependency
                # declared without code blocks associated to it.
                $global_named_blocks{$name} = [];
            }

            my $deps = $global_dependencies{$name};
            # I'm not sure why this script used to stop when no dependencies were declared.
            # stop("No dependencies declared for `$name`.") if !defined $deps;
            foreach(@{$deps->{cpp}}) {
                push @cpp_dependencies, $_ if !$seen_cpp{$_}++;
            }

            # Code blocks must be included *after* their dependencies, but to avoid double inclusions,
            # the named code blocks who are subsets of reffed codeblocks must be marked as seen.
            if(defined $reffed{$name}) {
                foreach(@{$reffed{$name}}) { $seen_noweb{$_}++; }
            }
            extract_dependencies(@{$deps->{noweb}}) if defined $deps->{noweb};
            push @noweb_dependencies, $name;
        }
    }
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
        my $linum_copy = $linum; # This language is driving me nuts.
        my $noweb_disabled = $global_lines[$linum_copy - 1] =~ /:noweb +no[$ ]/;

        until ((my $line = $global_lines[$linum_copy]) =~ /^\s*#\+end_src/ || $linum_copy >= @global_lines) {
            if ($line =~ /(\s*)<<(.+)>>/ && !$noweb_disabled) {
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
            ++$linum_copy;
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
};

###########################
# Putting it all together #
###########################
if($tangle) {
    while(my ($name, $destination) = each %global_tangle) {
        # Extract the dependencies *only* for the code block to tangle.
        @noweb_dependencies = ();
        extract_dependencies($name);

        make_necessary_dir($destination);
        open(my $dest_handle, '>:encoding(UTF-8)', $destination)
            or die "Failed to open `$1`.";
        select $dest_handle; # select STDOUT to restore STDOUT as the default output file.
        foreach(@noweb_dependencies) {
            print_codeblock($_);
        }
        close $dest_handle;
    }
    exit; # Tangling is done at the exclusion of anything else.
}

extract_dependencies(@$noweb);

foreach(@$cpp) {
    push @cpp_dependencies, $_ if !$seen_cpp{$_}++;
}

foreach(@cpp_dependencies) {
    say "#include <$_>";
}
foreach(@noweb_dependencies) {
    print_codeblock($_);
}
