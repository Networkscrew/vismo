# Contributing to vismo 👋

First of all thank you for reading this. It means you care about vismo. That matters a lot to me. 

Vismo is a tool that was built to solve a real problem. If you have ever been confused by a Kubernetes resource and wished something would explain it to you then you are the kind of person we are trying to help. Contributions of any size to vismo are welcome.

---

## I found a bug 🐛

If you find a bug in vismo open an issue. Tell us what you did what you expected to happen and what actually happened. If you can paste the output of level=debug that is a huge help. It shows every step that vismo took.

Do not worry about filing a bug that you think is dumb. If something confused you it is worth investigating and fixing.

---

## I have an idea 💡

If you have an idea for vismo open an issue. Describe it. You do not need to write a specification or make a prototype. A paragraph explaining the problem you want to solve is enough to start a conversation about this tool.

---

## I want to write code ⌨️

If you want to write code for vismo here is what you do:

1. Fork the vismo repository. Create a branch with a short name that describes what you are doing.

2. Make your change to vismo. Try to keep it focused on one thing.

3. Run make test and make build before you push your changes. If the tests pass and it compiles you are in shape.

4. Open a request with a short description of why you made the change to vismo.

### A few things that help us review your code

- If you add a detector or parser to vismo add a test for it.

- Log new operations with slog.Debug. Ensuring usage of --log-level=debug to remain useful for vismo.

- Follow the existing error style: fmt.Errorf.

- Do not add dependencies to vismo without a reason. We want to keep the small.

---

## I'm new to Go / Kubernetes 😥

That is okay. Vismo only uses a part of both Go and Kubernetes. If you want to contribute to vismo but feel unsure pick an issue and ask questions. I'm happy to help you with expanding vismo.

---

## What counts as a contribution 👍

Code is not the way to contribute to vismo. Improving an error message fixing a typo writing a test or sharing feedback on an issue. All of these things help vismo.

Thanks again, for being and considering contributing to `vismo`.

> I would really appreciate it if you could help me out by marking this vismo repository with a star ⭐. It will help vismo get attention. 💖