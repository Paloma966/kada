#!/usr/bin/env node
// Git message filter: drop an assistant attribution trailer from a commit message.
//
// Usage: git filter-branch -f --msg-filter "node <abs-path>/strip-coauthor-trailer.js" HEAD
//
// Reads the message on stdin and writes the cleaned message on stdout. Native sed is not reachable
// from every environment this repository is maintained in, and Node is already required by the
// tooling here.
//
// Use an absolute path for the script: git runs the filter from its own working directory, where a
// relative path may not resolve.

let input = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => {
  input += chunk;
});
process.stdin.on("end", () => {
  const cleaned = input
    .split("\n")
    .filter((line) => !/^Co-[Aa]uthored-[Bb]y:.*(claude|anthropic)/i.test(line.trim()))
    .join("\n")
    // git terminates the message with a newline; removing the trailer can leave the blank separator
    // line (or several blank lines) dangling at the end, which git would store as part of the body.
    .replace(/\n+$/, "\n");

  process.stdout.write(cleaned);
});
