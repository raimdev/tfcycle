# tfcycle - Terraform Cycle Error Analyzer

A powerful CLI tool that transforms frustrating Terraform cycle errors into clear, actionable analysis. No more staring at cryptic 50+ resource cycle lists trying to figure out what went wrong!

## Features

- 🔍 **Smart Parsing**: Handles complex cycle errors with modules, instance keys, and action annotations
- 🎯 **Minimal Cycle Detection**: Identifies the smallest cycles within large strongly connected components
- 💡 **Actionable Suggestions**: Provides specific recommendations based on resource types and patterns
- 📊 **Multiple Output Formats**: Human-readable text, JSON, and DOT visualization
- 🚀 **Fast & Reliable**: Built in Go for performance and cross-platform compatibility

## Installation

### Build from Source

```bash
git clone <repository>
cd tfcycle
go build -o tfcycle
```

### Usage

```bash
# Analyze error from terraform output
terraform plan 2>&1 | tfcycle analyze

# Analyze error from file
tfcycle analyze --error-file cycle_error.txt

# Generate DOT visualization
tfcycle visualize --output cycle.dot

# Verbose JSON output
tfcycle analyze --verbose --json

# Get help
tfcycle --help
```

## Examples

### Input (Typical Terraform Error)
```
Error: Cycle: module.ous.aws_organizations_organizational_unit.level1["dept1"], 
module.ous.aws_organizations_organizational_unit.level1["dept2"], 
module.bs_gr_audit.module.lambdacron_remove_shield.aws_iam_role.main (destroy), 
module.bs_gr_audit.module.lambdacron_remove_shield.aws_iam_role_policy_attachment.lambda_logs (destroy), 
module.bs_gr_audit.module.lambdacron_remove_shield.aws_iam_policy.lambda_logging (destroy), 
module.bs_gr_audit.module.lambdacron_remove_shield.aws_lambda_function.lambda_function (destroy), 
module.bs_gr_audit.module.lambdacron_remove_shield.aws_cloudwatch_event_rule.lambda_event_rule (destroy), 
module.bs_gr_audit.module.lambdacron_remove_shield.aws_cloudwatch_event_target.lambda_target (destroy), 
module.bs_gr_audit.module.lambdacron_remove_shield.aws_lambda_permission.allow_cloudwatch_to_call_check_foo (destroy), 
module.bs_gr_audit.aws_organizations_account.audit (destroy deposed f2ca8b5c)
```

### Output (Clear Analysis)
```
🔄 TERRAFORM CYCLE DETECTED

Minimal Cycle #1 (2 resources):
  1. module.ous.aws_organizations_organizational_unit.level1[dept1]
     ↳ depends on module.ous.aws_organizations_organizational_unit.level1[dept2]
  2. module.ous.aws_organizations_organizational_unit.level1[dept2]
     ↳ depends on module.ous.aws_organizations_organizational_unit.level1[dept1]

Minimal Cycle #2 (2 resources):
  1. module.bs_gr_audit.module.lambdacron_remove_shield.aws_iam_role.main (destroy)
     ↳ depends on module.bs_gr_audit.module.lambdacron_remove_shield.aws_iam_role_policy_attachment.lambda_logs
  2. module.bs_gr_audit.module.lambdacron_remove_shield.aws_iam_role_policy_attachment.lambda_logs (destroy)
     ↳ depends on module.bs_gr_audit.module.lambdacron_remove_shield.aws_iam_role.main

💡 SUGGESTIONS:
  • Break circular dependencies by removing direct references
  • Use data sources to reference existing resources
  • Consider splitting resources across multiple Terraform runs

🔧 COMMON SOLUTIONS:
  • Use lifecycle { create_before_destroy = true } for replacement scenarios
  • Replace direct references with data source lookups
  • Split complex resources into multiple Terraform configurations
  • Use depends_on explicitly to control dependency order
```

## Supported Input Formats

- Simple cycles: `aws_security_group.sg1, aws_security_group.sg2`
- Module paths: `module.vpc.aws_security_group.sg1`
- Instance keys: `aws_instance.web["key1"]`, `aws_instance.web[0]`
- Action annotations: `(destroy)`, `(expand)`, `(close)`, `(destroy deposed abc123)`
- Multi-line formatted errors
- Complex combinations of all above

## Architecture

The tool consists of several key components:

- **Parser**: Robust regex-based parsing of Terraform error messages
- **Analyzer**: Graph-based cycle detection and analysis
- **Formatter**: Multiple output formats (text, JSON, DOT)
- **CLI**: Command-line interface with comprehensive options

## Development

### Running Tests

```bash
go test -v
```

### Building

```bash
go build -o tfcycle
```

## License

[License information]

## Contributing

[Contributing guidelines]