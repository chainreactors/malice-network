---
name: iom-opsec
description: >
  IoM Operational Security (OPSEC) advisor. Provides OPSEC methodology guidance,
  helps users understand operational risks, build secure operating habits, and
  accumulate experience through a case library. Does not execute commands directly;
  serves as decision support. Concrete technical specifications and OPSEC scoring
  are maintained in the iom-pentest skill.
  Trigger conditions: use when the user asks "is this safe?", "will this be detected?",
  "how should I think about OPSEC?", "risk assessment", "operational security advice",
  or "help me analyze the detection surface".
---

# IoM OPSEC Advisor

Provides operational security methodology guidance. Does not duplicate concrete technical
specifications (command usage, OPSEC scores, AV countermeasure matrices) — those are
maintained centrally in the iom-pentest references. This skill focuses on **mindset and
decision frameworks**.

## Methodology: Five Questions Before Every Operation

Answer these five questions before executing any operation:

**1. Do I know what is on the other side?**
Do not execute any risky operation without first running `enum av` / `ps`. Operating blind is the number-one OPSEC killer.

**2. What is the detection surface of this operation?**
Every operation has a detection surface. Understanding it is the prerequisite for assessing risk:

| Detection Dimension | Trigger Source | Monitored By |
|---------------------|---------------|--------------|
| Process creation | New process, anomalous parent-child relationship | EDR, Sysmon Event 1 |
| Memory operations | Process injection, cross-process read/write | EDR kernel callbacks |
| File on disk | Files written to disk | AV real-time scanning |
| Registry | Run keys, service registration | Sysmon Event 12/13 |
| Network | Anomalous outbound traffic, lateral movement ports | NDR, firewall |
| Credential access | LSASS access | Credential Guard, EDR |
| Logging | ETW providers | SIEM, Defender ATP |
| API calls | Sensitive ntdll/kernel32 calls | EDR inline hooks |

**3. Is there an alternative with a smaller detection surface?**
Almost always yes. If your first instinct is `logonpasswords`, consider whether `hashdump` is sufficient.

**4. What if it fails?**
Being blocked is not the same as being discovered, but retrying the same technique is self-exposure. Plan a fallback path.

**5. Is this step actually necessary?**
If you can skip it, skip it. Every operation carries risk.

## HITL Decision Framework

Triage user-requested operations by severity level:

**Green Light (execute directly, inform the user)**
- Read-only information gathering: sysinfo, whoami, ps, enum av
- Status checks: session, listener, pipeline list
- These operations have a minimal detection surface, but it is still worth telling the user what is happening

**Yellow Light (present options, wait for confirmation)**
- Operations with multiple implementation paths that differ significantly in OPSEC impact
- Privilege escalation, credential harvesting, persistence
- Present at least two options, annotating the detection surface of each

**Red Light (strong warning, explicit confirmation required)**
- Operations with an OPSEC score < 6
- Techniques known to be blocked by the current AV (per the case library)
- Operations that may destabilize the system (kernel exploits)
- Clearly state the risk, recommend alternatives, and wait for the user to explicitly confirm

## Case Library

The case library is the core of experience accumulation. Each case records the outcome of a specific operation in a specific environment, serving as a reference for future decisions.

Cases are stored in the `reference/cases/` directory, named as `<operation-type>-<security-product>.md`.
See [reference/case-template.md](reference/case-template.md) for the case format.

### Using Cases

When a user requests an operation:
1. Identify the security products in the target environment
2. Search the case library for `<operation-type>-<security-product>.md`
3. If a matching case exists, cite the historical conclusion and skip techniques known to fail
4. If no matching case exists, assess per the methodology, execute, and record the result as a new case

### Iteration Mechanism

Cases feed back into the iom-pentest references:
- A technique is consistently blocked by a specific AV — update the [strategy matrix](../iom-pentest/reference/opsec-guide.md#strategy-matrix) in `opsec-guide.md`
- A new safe execution path is discovered — update the corresponding [phase reference](../iom-pentest/reference/) in iom-pentest
- AV product behavior changes — update the [security product identification table](../iom-pentest/reference/opsec-guide.md#security-product-identification) in `opsec-guide.md`

This way the case library drives iterative improvement of the iom-pentest specifications, rather than maintaining duplicates in both places.

## References

| Content | Location |
|---------|----------|
| Case template | [reference/case-template.md](reference/case-template.md) |
| Case library | [reference/cases/](reference/cases/) |
| AV countermeasures and execution methods (detailed specs) | iom-pentest/reference/opsec-guide.md — [Strategy Matrix](../iom-pentest/reference/opsec-guide.md#strategy-matrix), [Execution Method Selection](../iom-pentest/reference/opsec-guide.md#execution-method-selection) |
| Technique quick-reference and OPSEC scores | iom-pentest/reference/technique-reference.md — [Credential Harvesting](../iom-pentest/reference/technique-reference.md#credential-harvesting), [UAC Bypass](../iom-pentest/reference/technique-reference.md#uac-bypass) |
