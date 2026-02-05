write a full spec for this project. include initializing the git repo and writing a readme w/ mit license:

include research tasks for figuring out how claude code and codex surface usage or how we can get to that information. include the cron or otherwise system of waking the nightshift to work. include a very basic flow. cron (or similar) wakes up -> nightwatch builds a prompt for the orchestrator from config and budget (or takes the default. requires at least a project), tells the orchestrator to choose a task, assign it so it’s not picked up by another agent, and spawn sub agents to plan, implement, review (with sub-loops if needed). then, the next hour (config interval)  in the night it wakes up again and checks token budget. if it’s within threshold, the process repeats.  all is logged. defaults should never include using more than 10% of the day’s budget (or similar, so nobody get surprised by running with defaults and waking up broke. 

This is a voice transcribed note for an idea for a project called Night Watch. It will be a Go CLI tool with the configuration file that can be either YAML or JSON. And what the purpose of it is, is that it runs every night and does maintenance or other tasks on your code base, and you wake up in the morning to find to see what it's done. So it should feel like a surprising morning wake-up gift in a way that'll be the marketing angle anyway it should be fun. The CLI to configure and run it should be nice, with a strong focus on DX.

It will be meant for you developers who subscribe to coding agent plans where they have a weekly allotment of credits or budget and they want to make sure that they use it every week so they can configure it to either spend a certain to uh to allot for a certain percentage at the end of every day. So if during the day they coded for, you know one for say three-fourths of their daily budget if you take the week and divide it by you know take the take the weekly budget and divide it by seven Then it could spend the rest of the average, or they could say if it's getting towards the end of the week to spend more, or if it's the last day of the weekly budget to spend all the rest of it, or if they're not using a pay-by-week plan, they could set a dollar amount or a token amount. It should be fairly configurable. That would be the whole trick to this. Would be making sure that it doesn't spend too much or too little. Anyway, once they've configured it, they can either accept the default list of things that Nightwatch can do, which would be like code reviews, updating documentation, doing security checks, and it could run the gamut from creating PRs to just creating a report to doing nothing if it can't find anything useful. I have a list of things that it could do that I'll provide you. And then the other aspect to it is that it's not per project. Well, it can be for one project, like you could have a.nightwatch file in a project but you could also have it run across multiple projects so like I maintain a few open source libraries it could work across those libraries. And it could take into account the task list if there was one. And we could have different ways of providing those, or it could take into account the claude.md or agents.md markdown files. We could give it various ways of having input, ranging from it, just does whatever it's going to do by default to you can give it as much guidance as you want to, but most of the guidance should be. I was going to say it should be imperative, like code directing it to a task list or something, but maybe we could even have prompt guidance in the config. I'm not sure. 

Here's the list of stuff it could do. To start with anyway. 

“It’s done — here’s the PR”

Fully formed, review-ready artifacts. These should be boring in the best way.

linter
bug finder
auto DRY
refactor suggestor
API contract verifier
backward-compatibility checker
build time optimizer
documentation backfiller
commit message normalizer
changelog synthesizer
release note drafter
ADR drafter

These only belong here if they can meet a hard bar: deterministic, minimal diff, explainable rationale. If they can’t, they get demoted.

⸻

“Here’s what I found”

Completed analysis with conclusions, but no code touched. You’re deciding whether to act.

doc drift detector
semantic diff explainer
dead code detector
dependency risk scanner
test gap finder
test flakiness analyzer
logging quality auditor
metrics coverage analyzer
performance regression spotter
cost attribution estimator
security foot-gun finder
PII exposure scanner
privacy policy consistency checker
schema evolution advisor
event taxonomy normalizer
roadmap entropy detector
bus-factor analyzer
knowledge silo detector

This is your morning briefing. Skimmable, ranked, opinionated.

⸻

“Here are options — what do you want to do?”

These surface judgment calls, tradeoffs, or design forks. The system shouldn’t decide, but it should think.

task groomer
guide / skill improver
idea generator
tech-debt classifier
“why does this exist?” annotator
edge-case enumerator
error-message improver
SLO / SLA candidate suggester
UX copy sharpener
accessibility linting (non-checkbox)
“should this be a service?” advisor
ownership boundary suggester
oncall load estimator

The quality bar here is framing: crisp options, explicit tradeoffs, implied cost of inaction.

⸻

“I tried it safely”

Work that required execution or simulation but left no lasting side effects.

migration rehearsal runner
integration contract fuzzer
golden-path recorder
performance profiling runs
allocation / hot-path profiling

These reduce uncertainty before you commit. Think rehearsal, not action.

⸻

“Here’s the map”

Pure context you didn’t have yesterday, now laid out cleanly.

visibility instrumentor
repo topology visualizer
permissions / auth surface mapper
data lifecycle tracer
feature flag lifecycle monitor
CI signal-to-noise scorer
historical context summarizer

This is slow-burn value. Over time it changes how fast you reason.

⸻

“For when things go sideways”

Artifacts you hope to never need, but are grateful exist.

runbook generator
rollback plan generator
incident postmortem draft generator

These should age quietly until the day they save you hours.

⸻



No output is “done” unless it answers: What would you like to do next?

Sometimes the answer is “nothing — merged.”
Sometimes it’s “pick A or B.”
Sometimes it’s “here’s the context you were missing.”


