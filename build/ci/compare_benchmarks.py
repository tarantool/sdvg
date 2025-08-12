import re
import argparse
import statistics
import pandas as pd
from typing import Dict, Union, Tuple, List, Literal

KNOWN_METRICS = {
    'ns/op':    {'better': 'down'},
    'MB/s':     {'better': 'up'},
    'rows/s':   {'better': 'up'},
    'values/s': {'better': 'up'},
}

TRACKED_METRICS = {'MB/s', 'values/s'}

EMOJIS = {
    'good': '⚡️',
    'bad': '💔'
}


def is_bench_line(line: str) -> bool:
    return line.startswith("Benchmark")


def parse_bench_line(line: str) -> Union[Tuple[str, Dict[str, float]], None]:
    """parses `go test -bench` results output.
    Example:

    BenchmarkPartitioning/CI/cpu-4          2569041    475.5 ns/op    218.73 MB/s    8412793 rows/s   16825587 values/s

    result:
    ('BenchmarkPartitioning/CI/cpu-4', {'ns/op': 475.5, 'MB/s': 218.73, 'rows/s': 8412793, 'values/s': 16825587}
    """

    parts = re.split(r'\s+', line.strip())
    if len(parts) < 3 or not is_bench_line(parts[0]):
        return None

    bench_name = parts[0]

    metrics = {}
    for value, metric in zip(parts[2::2], parts[3::2]):
        if metric not in KNOWN_METRICS:
            raise ValueError(f"Unknown metric '{metric}' in test '{bench_name}'")
        try:
            val = float(value)
        except ValueError:
            raise ValueError(f"Failed to parse value '{value}' for '{metric}'")
        metrics[metric] = val
    return bench_name, metrics


def parse_metrics_file(path: str) -> Dict[str, Dict[str, List[float]]]:
    results = {}
    with open(path) as f:
        for line in f:
            if not is_bench_line(line):
                continue
            parsed = parse_bench_line(line)
            if parsed is None:
                raise ValueError(f"failed parse Benchmark line: '{line}'")

            name_test, metrics = parsed
            if name_test not in results:
                results[name_test] = {m: [] for m in KNOWN_METRICS.keys()}

            for metric_name, value in metrics.items():
                results[name_test][metric_name].append(value)
    return results


def aggregate_results(parsed_metrics: Dict[str, Dict[str, List[float]]],
                      method: Literal["mean", "median"]) -> Dict[str, Dict[str, float]]:
    aggregated: Dict[str, Dict[str, float]] = {}
    for bench_name, metrics in parsed_metrics.items():
        aggregated[bench_name] = {}
        for m, values in metrics.items():
            if not values:
                continue
            if method == "median":
                aggregated[bench_name][m] = statistics.median(values)
            elif method == "mean":
                aggregated[bench_name][m] = statistics.mean(values)
    return aggregated


def format_metric_changes(
        old_val, new_val: float,
        metric_name: str,
        alert_threshold: float,
        alert_emojis: Dict[str, str],
        metric_change_interpretation: Dict[str, Dict[str, str]],
        only_one_file: bool = False,
) -> str:
    # the metric doesn't exist for this bench
    if old_val is None and new_val is None:
        raise f"both old and new values are 'None' for metric {metric_name}"

    if only_one_file:
        return humanize_number(old_val) if old_val else humanize_number(new_val)

    # the metric exists only in new file
    if old_val is None and new_val is not None:
        return f"New metric: {new_val:.2f} ⚠️"

    # the metric exists only in old file
    if new_val is None and old_val is not None:
        return f"Only old metric: {old_val:.2f} ⚠️"

    change_pct = ((new_val - old_val) / old_val) * 100

    better_direction = metric_change_interpretation[metric_name]['better']
    is_better = (change_pct > 0 if better_direction == 'up' else change_pct < 0)

    emoji = ""
    if alert_threshold is not None and abs(change_pct) >= alert_threshold:
        emoji = alert_emojis['good'] if is_better else alert_emojis['bad']

    res = f"{humanize_number(old_val)} → {humanize_number(new_val)} ({change_pct:+.2f}%)"
    return res if emoji == "" else res + f" {emoji}"


def humanize_number(val: float) -> str:
    abs_val = abs(val)
    if abs_val >= 1_000_000:
        return f"{val/1_000_000:.2f}M"
    elif abs_val >= 1_000:
        return f"{val/1_000:.2f}K"
    else:
        return f"{val:.2f}"


def format_benchmark_name(raw_name: str) -> str:
    name = raw_name[len("Benchmark"):] if raw_name.startswith("Benchmark") else raw_name
    parts = name.split("/")
    if len(parts) == 1:
        return parts[0]

    base_name = " ".join(parts[:-1])
    params_chunk = parts[-1]

    params_split = params_chunk.split("-")
    params = []

    for i in range(0, len(params_split) - 1, 2):
        params.append(f"{params_split[i]}={params_split[i+1]}")

    if params:
        return f"{base_name} ({', '.join(params)})"
    else:
        return f"{base_name}"


def compare_benchmarks_df(old_metrics, new_metrics, alert_threshold=None):
    only_one_file = (old_metrics is None) ^ (new_metrics is None)

    if old_metrics is None:
        old_metrics = {}

    if new_metrics is None:
        new_metrics = {}

    rows = []
    for bench_name in sorted(set(old_metrics.keys()) | set(new_metrics.keys())):
        if "/CI/" not in bench_name:
            continue
        row = {"Benchmark": format_benchmark_name(bench_name.replace("/CI/", "/"))}

        for metric in TRACKED_METRICS:
            old_val = old_metrics.get(bench_name, {}).get(metric, None)
            new_val = new_metrics.get(bench_name, {}).get(metric, None)

            try:
                formated_metric = format_metric_changes(
                    old_val, new_val, metric, alert_threshold, EMOJIS, KNOWN_METRICS,
                    only_one_file=only_one_file,
                )
                row[metric] = formated_metric
            except Exception as e:
                raise f"failed format metric changes for benchmark '{bench_name}': {e}"
        rows.append(row)
    df = pd.DataFrame(rows)
    df = df[["Benchmark"] + list(TRACKED_METRICS)]
    return df.to_markdown(index=False)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Compare go test -bench results in markdown format")
    parser.add_argument("--old-file", help="Path to old benchmark results file", default=None)
    parser.add_argument("--new-file", help="Path to new benchmark results file", default=None)
    parser.add_argument("--alert-threshold", type=float, default=5,
                        help="Percent change threshold for adding emoji alerts")
    parser.add_argument("--aggregation", choices=["mean", "median"], default="mean",
                        help="Aggregation method for multiple runs of the same benchmark")
    args = parser.parse_args()

    old_metrics = None
    new_metrics = None

    if not args.old_file and not args.new_file:
        parser.error("You must specify at least --old-file or --new-file")

    if args.old_file:
        old_metrics = aggregate_results(parse_metrics_file(args.old_file), args.aggregation)

    if args.new_file:
        new_metrics = aggregate_results(parse_metrics_file(args.new_file), args.aggregation)

    print(compare_benchmarks_df(old_metrics, new_metrics, alert_threshold=args.alert_threshold))
