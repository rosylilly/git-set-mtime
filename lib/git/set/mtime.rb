require 'thor'
require 'rugged'
require "git/set/mtime/version"

module Git::Set::Mtime
  class CLI < Thor
    desc 'apply', 'apply mtime to files'
    def apply
      repo = Rugged::Repository.new(Dir.pwd)
      head = repo.head
      target = head
      while target.respond_to?(:target)
        target = target.target
      end

      target.tree.each { |e| puts e }
    end

    default_task :apply
  end
end
