require 'pathname'
require 'tempfile'
require 'English'

MYDIR = Pathname(__FILE__).realpath.dirname
VERSION = `cd #{MYDIR} ; git describe`.strip

task :default => [:help]

desc 'Run the Moar tests'
task :test do
  ruby "#{MYDIR}/src/test.rb"
end

desc 'Print a help message'
task :help do
  puts `rake --tasks`
end

desc 'Install Moar system wide'
task :install => [:test]
task :install, :directory do |_t, args|
  args.with_defaults(:directory => '/usr/local/bin')
  destination_file = "#{args[:directory]}/moar"

  # Copy moar.rb into a temporary location, replacing the VERSION=
  # line with a fixed one
  Tempfile.open(['moar', '.rb']) do |tempfile|
    File.open("#{MYDIR}/src/moar.rb").each_line do |line|
      tempfile.puts(line.sub(/^VERSION *=.*/, "VERSION = '#{VERSION}'"))
    end
    tempfile.flush

    # Now, install the fixed-version file
    system('install', tempfile.path, destination_file) || exit(1)
  end

  puts "Installed into #{destination_file}"
end

# Return stdout from running the commandline, throw exception on failure
def run(commandline)
  output = `#{commandline}`
  raise "Command failed: #{commandline}" unless $CHILD_STATUS.success?
  return output
end

desc 'Make a release'
task :release => [:test]
task :release, :version do |_t, args|
  new_version = args[:version]
  raise 'A version number must be supplied' if new_version.nil?

  # FIXME: Verify we aren't dirty, can't make dirty releases

  # FIXME: Verify we're on the master branch, releases should come from master

  # Generate a message for the new version tag
  ANNOTATED_MSG = '/tmp/ANNOTATED_MSG'.freeze
  File.open(ANNOTATED_MSG, 'w') do |annotated_msg|
    annotated_msg.puts("Release #{new_version}")
    annotated_msg.puts
  end

  LAST_TAG = run('git describe --tags --abbrev=0').strip.freeze
  run("git log --no-decorate --first-parent #{LAST_TAG}..HEAD --oneline >> #{ANNOTATED_MSG}")

  # Make the annotated tag
  run("git tag --annotate -F #{ANNOTATED_MSG}")

  # Make a release build
  releasefile_name = "#{MYDIR}/moar-#{VERSION}.rb"
  File.open(releasefile_name, 'w') do |releasefile|
    File.open("#{MYDIR}/src/moar.rb").each_line do |line|
      releasefile.puts(line.sub(/^VERSION *=.*/, "VERSION = '#{VERSION}'"))
    end
  end

  puts
  puts "Release build written to #{releasefile_name}"
  puts

  run('git push --tags')

  puts 'Now go here...'
  puts "  https://github.com/walles/moar/releases/#{new_version}"
  puts '... click on "Edit tag", upload...'
  puts "  #{releasefile_name}"
  puts '... and click "Publish release"'
end
