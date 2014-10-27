require 'thor'
require 'time'
require "git/set/mtime/version"

module Git::Set::Mtime
  class CLI < Thor
    desc 'apply', 'apply mtime to files'
    def apply
      files = `git ls-files`
      files.each_line do |file|
        file = file.strip
        mtime_str = `git log -n 1 --date=local | head -n 3 | tail -n 1`.tr('Date:', '').strip
        mtime = Time.parse(mtime_str)
        File.utime(File.atime(file), mtime, file)
        puts "#{mtime} #{file}"
      end
    end

    default_task :apply
  end
end
