require 'thor'
require 'time'
require "git/set/mtime/version"

module Git::Set::Mtime
  class CLI < Thor
    desc 'apply', 'apply mtime to files'
    def apply
      files = `git ls-files`
      files.each_line do |file|
        file.chomp!
        mtime_str = `git log -1 --pretty='format:%ad' --date=local '#{file}'`
        mtime     = Time.parse(mtime_str)
        File.utime(File.atime(file), mtime, file)
        puts "#{mtime} #{file}"
      end
    end

    default_task :apply
  end
end
