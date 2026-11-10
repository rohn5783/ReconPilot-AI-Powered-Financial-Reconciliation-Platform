import React, { useMemo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  flexRender,
  createColumnHelper,
} from '@tanstack/react-table';
import { Asset, AssetDrift, Finding } from '../../types';
import { useWorkspaceStore } from '../../store/useWorkspaceStore';
import {
  Globe,
  Server,
  ShieldAlert,
  ArrowUpDown,
  ExternalLink,
  PlusCircle,
  AlertCircle,
} from 'lucide-react';

interface AssetInventoryViewProps {
  assets: Asset[];
  drifts: AssetDrift[];
  findings: Finding[];
}

const columnHelper = createColumnHelper<Asset>();

export const AssetInventoryView: React.FC<AssetInventoryViewProps> = ({
  assets,
  drifts,
  findings,
}) => {
  const { searchQuery, selectedAsset, setSelectedAsset, setSelectedFinding } = useWorkspaceStore();

  // Create lookup for drift badges by FQDN
  const driftMap = useMemo(() => {
    const map = new Map<string, AssetDrift[]>();
    for (const d of drifts) {
      const list = map.get(d.fqdn) || [];
      list.push(d);
      map.set(d.fqdn, list);
    }
    return map;
  }, [drifts]);

  // Create lookup for findings count by FQDN
  const findingMap = useMemo(() => {
    const map = new Map<string, Finding[]>();
    for (const f of findings) {
      const list = map.get(f.fqdn) || [];
      list.push(f);
      map.set(f.fqdn, list);
    }
    return map;
  }, [findings]);

  // Filtered dataset
  const filteredData = useMemo(() => {
    if (!searchQuery) return assets;
    const q = searchQuery.toLowerCase();
    return assets.filter((a) => {
      const matchFQDN = a.fqdn.toLowerCase().includes(q);
      const matchTitle = a.page_title.toLowerCase().includes(q);
      const matchIPs = a.ip_addresses?.some((ip) => ip.includes(q));
      const matchPorts = a.open_ports?.some((p) => p.toString().includes(q));
      const matchTech = a.tech_stack?.some((t) => t.toLowerCase().includes(q));
      return matchFQDN || matchTitle || matchIPs || matchPorts || matchTech;
    });
  }, [assets, searchQuery]);

  const columns = useMemo(
    () => [
      columnHelper.accessor('fqdn', {
        header: ({ column }) => (
          <button
            onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
            className="flex items-center gap-1 font-mono uppercase tracking-wider text-[11px] text-typography-muted hover:text-typography-primary"
          >
            <span>Target Host / FQDN</span>
            <ArrowUpDown className="w-3 h-3" />
          </button>
        ),
        cell: (info) => {
          const fqdn = info.getValue();
          const hostDrifts = driftMap.get(fqdn) || [];
          const hostFindings = findingMap.get(fqdn) || [];
          const hasCritical = hostFindings.some((f) => f.severity === 'critical');
          const hasHigh = hostFindings.some((f) => f.severity === 'high');

          return (
            <div className="flex flex-col py-1 space-y-1">
              <div className="flex items-center gap-2">
                <Globe className="w-3.5 h-3.5 text-signal-accent shrink-0" />
                <span className="font-mono font-semibold text-typography-primary tracking-tight">
                  {fqdn}
                </span>
                <a
                  href={`https://${fqdn}`}
                  target="_blank"
                  rel="noreferrer"
                  onClick={(e) => e.stopPropagation()}
                  className="text-typography-dim hover:text-signal-accent transition-colors"
                >
                  <ExternalLink className="w-3 h-3" />
                </a>
              </div>

              {/* Badges for Drift Mutations and Vulnerability Flags */}
              <div className="flex flex-wrap items-center gap-1 font-mono text-[10px]">
                {hostDrifts.map((d) => (
                  <span
                    key={d.id}
                    className="px-1.5 py-0.2 rounded-micro border border-signal-high/40 bg-signal-highBg text-signal-high font-medium"
                  >
                    {d.change_type === 'new_host'
                      ? '+ NEW HOST'
                      : d.change_type === 'port_opened'
                      ? `+ PORT ${d.details?.port} OPEN`
                      : d.change_type.toUpperCase()}
                  </span>
                ))}

                {hasCritical && (
                  <span className="px-1.5 py-0.2 rounded-micro border border-signal-critical/40 bg-signal-criticalBg text-signal-critical font-bold flex items-center gap-0.5">
                    <ShieldAlert className="w-2.5 h-2.5" />
                    CRITICAL RISK
                  </span>
                )}
                {hasHigh && !hasCritical && (
                  <span className="px-1.5 py-0.2 rounded-micro border border-signal-high/40 bg-signal-highBg text-signal-high font-semibold">
                    HIGH RISK
                  </span>
                )}
              </div>
            </div>
          );
        },
      }),

      columnHelper.accessor('http_status', {
        header: 'HTTP Status',
        cell: (info) => {
          const status = info.getValue();
          let color = 'bg-signal-nominalBg border-signal-nominal/40 text-signal-nominal';
          if (status >= 400 && status < 500) {
            color = 'bg-signal-mediumBg border-signal-medium/40 text-signal-medium';
          } else if (status >= 500) {
            color = 'bg-signal-criticalBg border-signal-critical/40 text-signal-critical';
          } else if (status === 0 || !status) {
            color = 'bg-carbon-card border-carbon-border text-typography-muted';
          }

          return (
            <div className="flex items-center gap-2">
              <span className={`px-2 py-0.5 rounded-micro border font-mono text-xs font-semibold ${color}`}>
                {status || 'TCP'}
              </span>
              <span className="text-xs text-typography-muted truncate max-w-[140px]" title={info.row.original.page_title}>
                {info.row.original.page_title || '—'}
              </span>
            </div>
          );
        },
      }),

      columnHelper.accessor('ip_addresses', {
        header: 'Resolved IPs & ASN',
        cell: (info) => {
          const ips = info.getValue() || [];
          const asn = info.row.original.asn;
          return (
            <div className="flex flex-col font-mono text-xs">
              <span className="text-typography-primary">
                {ips.length > 0 ? ips.join(', ') : '—'}
              </span>
              {asn && (
                <span className="text-[10px] text-typography-dim">
                  {asn} {info.row.original.asn_org ? `(${info.row.original.asn_org})` : ''}
                </span>
              )}
            </div>
          );
        },
      }),

      columnHelper.accessor('open_ports', {
        header: 'Open Ports',
        cell: (info) => {
          const ports = info.getValue() || [];
          return (
            <div className="flex flex-wrap items-center gap-1 font-mono text-xs">
              {ports.map((p) => {
                const isHighRisk = [22, 6379, 3306, 5432, 27017, 9200].includes(p);
                return (
                  <span
                    key={p}
                    className={`px-1.5 py-0.2 rounded-micro border text-[11px] font-semibold ${
                      isHighRisk
                        ? 'border-signal-critical/50 bg-signal-criticalBg text-signal-critical'
                        : 'border-carbon-border bg-carbon-card text-typography-secondary'
                    }`}
                  >
                    {p}
                  </span>
                );
              })}
            </div>
          );
        },
      }),

      columnHelper.accessor('tech_stack', {
        header: 'Fingerprinted Tech Stack',
        cell: (info) => {
          const techs = info.getValue() || [];
          return (
            <div className="flex flex-wrap items-center gap-1 text-[11px]">
              {techs.map((t) => (
                <span
                  key={t}
                  className="px-1.5 py-0.2 rounded-micro border border-carbon-border bg-carbon-card/70 text-typography-secondary font-mono"
                >
                  {t}
                </span>
              ))}
            </div>
          );
        },
      }),
    ],
    [driftMap, findingMap]
  );

  const table = useReactTable({
    data: filteredData,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-carbon-canvas">
      {/* Subheader Filter Bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-carbon-panel hairline-b text-xs text-typography-muted">
        <div className="flex items-center gap-3">
          <span className="font-mono text-typography-primary font-semibold">
            {filteredData.length} Discovered Hosts
          </span>
          <span className="text-typography-dim">•</span>
          <span className="font-mono text-typography-secondary">
            {drifts.length} Recorded Mutations
          </span>
          <span className="text-typography-dim">•</span>
          <span className="font-mono text-typography-secondary">
            {findings.length} Vulnerability Findings
          </span>
        </div>

        <span className="text-[11px] font-mono text-typography-dim">
          Click any host row to inspect raw HTTP headers & audit evidence
        </span>
      </div>

      {/* TanStack Table Grid */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-left border-collapse">
          <thead className="bg-carbon-panel hairline-b sticky top-0 z-10 select-none">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="px-4 py-2.5 text-[11px] font-mono text-typography-muted font-semibold tracking-wider hairline-r"
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>

          <tbody className="divide-y divide-carbon-border">
            {table.getRowModel().rows.map((row) => {
              const isSelected = selectedAsset?.id === row.original.id;
              return (
                <tr
                  key={row.id}
                  onClick={() => {
                    setSelectedAsset(row.original);
                    const hostFindings = findingMap.get(row.original.fqdn);
                    if (hostFindings && hostFindings.length > 0) {
                      setSelectedFinding(hostFindings[0]);
                    } else {
                      setSelectedFinding(null);
                    }
                  }}
                  className={`cursor-pointer transition-colors ${
                    isSelected
                      ? 'bg-carbon-active border-l-2 border-l-signal-accent'
                      : 'hover:bg-carbon-hover/60'
                  }`}
                >
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="px-4 py-2.5 hairline-r">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};

