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

desc 'Make a release build'
task :release => [:test]
task :release do
  releasefile_name = "#{MYDIR}/moar-#{VERSION}.rb"
  File.open(releasefile_name, 'w') do |releasefile|
    File.open("#{MYDIR}/src/moar.rb").each_line do |line|
      releasefile.puts(line.sub(/^VERSION *=.*/, "VERSION = '#{VERSION}'"))
    end
  end

  puts
  puts "Release build written to #{releasefile_name}"
end
