import argparse
import re
import statistics
import os
import textwrap
from collections import OrderedDict
from typing import Dict, Tuple, List, Literal, Optional

import pandas as pd

METRICS = {
    'MB/s': {'name': 'B/s', 'good_direction': 'up', 'scale': 2 ** 20},
    'values/s': {'good_direction': 'up'},
    # 'ns/op': {'name': 's/op', 'good_direction': 'down', 'scale': 1e-9},
    # 'rows/s': {'good_direction': 'up'},
}

EMOJIS = {
    'good': '⚡️',
    'bad': '💔'
}


def format_benchmark_name(name: str) -> str:
    name = name.replace("Benchmark", "")
    name = name.replace("/CI/", "/")

    parts = name.split("/")
    if len(parts) == 1:
        return parts[0]

    base_name = " ".join(parts[:-1])
    params_split = parts[-1].split("-")

    params = []
    for i in range(0, len(params_split) - 1, 2):
        params.append(f"{params_split[i]}={params_split[i + 1]}")

    if params:
        return f"{base_name} ({', '.join(params)})"
    else:
        return base_name


def parse_bench_line(line: str) -> Tuple[Optional[str], Optional[Dict[str, float]]]:
    """parses `go test -bench` results output.
    Example:

    BenchmarkPartitioning/CI/cpu-4          2569041    475.5 ns/op    218.73 MB/s    8412793 rows/s   16825587 values/s

    result:
    ('Partitioning (cpu=4)', {'ns/op': 475.5, 'MB/s': 218.73, 'rows/s': 8412793, 'values/s': 16825587}
    """

    parts = re.split(r'\s+', line.strip())
    if len(parts) < 3 or not parts[0].startswith("Benchmark") or "/CI/" not in parts[0]:
        return None, None

    bench_name = format_benchmark_name(parts[0])

    metrics = {}
    for value, metric in zip(parts[2::2], parts[3::2]):
        if metric not in METRICS:
            continue
        try:
            metrics[metric] = float(value)
        except ValueError:
            raise ValueError(f"Failed to parse value '{value}' for '{metric}'")

    return bench_name, metrics


def parse_metrics_file(path: str) -> Dict[str, Dict[str, List[float]]]:
    results = {}

    with open(path) as f:
        for line in f:
            name_test, metrics = parse_bench_line(line)
            if name_test is None:
                continue

            if not metrics:
                continue

            if name_test not in results:
                results[name_test] = {m: [] for m in METRICS.keys()}

            for metric_name, value in metrics.items():
                results[name_test][metric_name].append(value)

    return results


def aggregate_results(
        parsed_metrics: Dict[str, Dict[str, List[float]]],
        method: Literal["mean", "median"]
) -> OrderedDict[str, Dict[str, float]]:
    aggregated: OrderedDict[str, Dict[str, float]] = OrderedDict()

    for bench_name, metrics in parsed_metrics.items():
        aggregated[bench_name] = {}

        for m, values in metrics.items():
            if method == "median":
                aggregated[bench_name][m] = statistics.median(values)
            elif method == "mean":
                aggregated[bench_name][m] = statistics.mean(values)

    return aggregated


def humanize_number(val: float, scale: float) -> str:
    if val is None:
        return "?"

    val = val * scale
    abs_val = abs(val)
    if abs_val >= 1_000_000:
        return f"{val / 1_000_000:.2f}M"
    elif abs_val >= 1_000:
        return f"{val / 1_000:.2f}K"
    else:
        return f"{val:.2f}"


def format_metric_changes(metric_name: str, old_val, new_val: Optional[float], alert_threshold: float) -> str:
    old_val_str = humanize_number(old_val, METRICS[metric_name].get('scale', 1))
    new_val_str = humanize_number(new_val, METRICS[metric_name].get('scale', 1))

    if old_val is None or new_val is None:
        suffix = " ⚠️"
    else:
        change_pct = (new_val / old_val - 1) * 100
        suffix = f" ({change_pct:+.2f}%)"

        if abs(change_pct) >= alert_threshold:
            is_better = METRICS[metric_name].get('good_direction') == 'up' and change_pct > 0
            suffix += f" {EMOJIS['good'] if is_better else EMOJIS['bad']}"

    return f"{old_val_str} → {new_val_str}{suffix}"


def compare_benchmarks_df(old_metrics, new_metrics, alert_threshold=None):
    if old_metrics is None:
        old_metrics = {}

    if new_metrics is None:
        new_metrics = {}

    all_metrics = OrderedDict()
    all_metrics.update(old_metrics)
    all_metrics.update(new_metrics)

    df = pd.DataFrame(columns=["Benchmark"] + [v.get('name', k) for k, v in METRICS.items()])

    for bench_name in all_metrics.keys():
        row = {"Benchmark": bench_name}

        for metric_name, metric_params in METRICS.items():
            old_val = old_metrics.get(bench_name, {}).get(metric_name, None)
            new_val = new_metrics.get(bench_name, {}).get(metric_name, None)
            row[metric_params.get('name', metric_name)] = format_metric_changes(
                metric_name, old_val, new_val, alert_threshold
            )

        df.loc[len(df)] = row

    return df.to_markdown(index=False)


def build_report_header(old_file, sha_file: str) -> str:
    event_name = os.environ.get("GITHUB_EVENT_NAME", "")
    base_branch = os.environ.get("GITHUB_DEFAULT_BRANCH", "master")

    warning = ""
    if not os.path.exists(old_file):
        warning = textwrap.dedent("""
            > [!WARNING]
            > No test results found for master branch. Please run workflow on master first to compare results.
        """).strip()

    if event_name == "pull_request":
        pr_branch = os.environ.get("GITHUB_HEAD_REF", "")
        header_ending = f"`{pr_branch}`" if not os.path.exists(old_file) else f"`{base_branch}` VS `{pr_branch}`"
    else:
        if not os.path.exists(old_file):
            header_ending = f"`{base_branch}`"
        else:
            prev_master_sha = "(sha not found)"
            if sha_file and os.path.exists(sha_file):
                with open(sha_file) as f:
                    prev_master_sha = f.read().strip()

            commit_sha = os.environ.get("GITHUB_SHA", "")[:7]
            header_ending = f"`{base_branch} {prev_master_sha}` VS `{base_branch} {commit_sha}`"

    header = f"# Perf tests report: {header_ending}\n"
    return f"{warning}\n\n{header}" if warning else header


def main():
    parser = argparse.ArgumentParser(description="Compare go test -bench results in markdown format")
    parser.add_argument(
        "--alert-threshold", type=float, default=7,
        help="Percent change threshold for adding emoji alerts"
    )
    parser.add_argument(
        "--aggregation", choices=["mean", "median"], default="mean",
        help="Aggregation method for multiple runs of the same benchmark"
    )
    parser.add_argument("--old-commit-sha-path", help="Path to file with sha commit of the old benchmark")
    parser.add_argument("old_file", help="Path to old benchmark results file", nargs='?', default="")
    parser.add_argument("new_file", help="Path to new benchmark results file")
    args = parser.parse_args()

    old_metrics = None
    if args.old_file and os.path.exists(args.old_file):
        old_metrics = aggregate_results(parse_metrics_file(args.old_file), args.aggregation)

    new_metrics = aggregate_results(parse_metrics_file(args.new_file), args.aggregation)

    print(build_report_header(args.old_file, args.old_commit_sha_path))
    print(compare_benchmarks_df(old_metrics, new_metrics, alert_threshold=args.alert_threshold))


if __name__ == "__main__":
    main()
