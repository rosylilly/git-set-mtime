require 'thor'
require 'time'
require 'git/set/mtime/version'
require 'open3'

module Git::Set::Mtime
  class CLI < Thor
    GIT_LOG_ARGS = %w[git log -1 --pretty=format:%ad --date local].freeze

    desc 'apply', 'apply mtime to files'
    def apply
      files = `git ls-files`
      files.each_line do |file|
        file.chomp!
        mtime_str, status = Open3.capture2e(*GIT_LOG_ARGS, file)
        raise mtime_str unless status.success?
        mtime = Time.parse(mtime_str)
        File.utime(File.atime(file), mtime, file)
        puts "#{mtime} #{file}"
      end
    end

    default_task :apply
  end
end
