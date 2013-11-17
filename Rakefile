require 'pathname'
require 'tempfile'

MYDIR = Pathname(__FILE__).realpath.dirname
VERSION = `cd #{MYDIR} ; git describe`.strip

task :default => [:help]

desc 'Run the Moar tests'
task :test do
  ruby "#{MYDIR}/src/test.rb"
end

desc 'Print a help message'
task :help do
  puts <<eos
To run the unit tests:
  rake test

To install in /usr/local/bin:
  sudo rake install

To install in /usr/bin:
  sudo rake install[/usr/bin]
eos
end

desc 'Install Moar system wide'
task :install => [:test]
task :install, :directory do |t, args|
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
