import React, { useState } from 'react';
import { AIReport, ScanJob } from '../../types';
import { api } from '../../api/client';
import {
  ShieldAlert,
  Terminal,
  Cpu,
  Download,
  Copy,
  Check,
  RefreshCw,
  AlertTriangle,
  ArrowUpRight,
  ShieldCheck,
  FileText,
  Activity,
  Layers,
  Sparkles,
} from 'lucide-react';

interface AIAnalystViewProps {
  report: AIReport | null;
  scanJob: ScanJob | null;
  onRefreshReport?: () => void;
}

export const AIAnalystView: React.FC<AIAnalystViewProps> = ({
  report,
  scanJob,
  onRefreshReport,
}) => {
  const [copiedSnippetIndex, setCopiedSnippetIndex] = useState<number | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [selectedPlaybookCategory, setSelectedPlaybookCategory] = useState<string>('all');

  const handleCopySnippet = (snippet: string, index: number) => {
    navigator.clipboard.writeText(snippet);
    setCopiedSnippetIndex(index);
    setTimeout(() => setCopiedSnippetIndex(null), 2000);
  };

  const handleRegenerate = async () => {
    if (!scanJob) return;
    setIsGenerating(true);
    try {
      await api.generateReport(scanJob.id);
      if (onRefreshReport) {
        onRefreshReport();
      }
    } catch (err) {
      console.error('Failed to regenerate report:', err);
    } finally {
      setIsGenerating(false);
    }
  };

  if (!report) {
    return (
      <div className="h-full flex flex-col items-center justify-center p-8 text-center bg-[#08090C]">
        <div className="w-12 h-12 rounded border border-[#1E222D] bg-[#0D0F14] flex items-center justify-center mb-4 text-[#8A90A2]">
          <Cpu className="w-6 h-6 animate-pulse text-[#38BDF8]" />
        </div>
        <h3 className="text-sm font-mono font-medium text-[#E1E4EA] mb-1">
          NO SYNTHESIS REPORT GENERATED
        </h3>
        <p className="text-xs text-[#8A90A2] max-w-md mb-6">
          The autonomous security synthesis engine generates structured executive briefs,
          drift calculations, and copyable defensive playbooks upon scan completion.
        </p>
        {scanJob && (
          <button
            onClick={handleRegenerate}
            disabled={isGenerating}
            className="flex items-center space-x-2 px-3 py-1.5 bg-[#38BDF8]/10 hover:bg-[#38BDF8]/20 border border-[#38BDF8]/40 text-[#38BDF8] text-xs font-mono rounded transition-colors disabled:opacity-50"
          >
            <Sparkles className={`w-3.5 h-3.5 ${isGenerating ? 'animate-spin' : ''}`} />
            <span>{isGenerating ? 'SYNTHESIZING WITH GEMINI...' : 'TRIGGER AI SYNTHESIS'}</span>
          </button>
        )}
      </div>
    );
  }

  const riskScore = report.overall_risk_score ?? 0;
  const getRiskScoreColor = (score: number) => {
    if (score >= 75) return { text: 'text-[#F43F5E]', bar: 'bg-[#F43F5E]', border: 'border-[#F43F5E]/30', label: 'CRITICAL' };
    if (score >= 50) return { text: 'text-[#F97316]', bar: 'bg-[#F97316]', border: 'border-[#F97316]/30', label: 'ELEVATED' };
    if (score >= 25) return { text: 'text-[#EAB308]', bar: 'bg-[#EAB308]', border: 'border-[#EAB308]/30', label: 'MODERATE' };
    return { text: 'text-[#10B981]', bar: 'bg-[#10B981]', border: 'border-[#10B981]/30', label: 'NOMINAL' };
  };

  const riskColor = getRiskScoreColor(riskScore);

  const categories = [
    'all',
    ...Array.from(new Set(report.remediation_playbook?.map((item) => item.target_category) || [])),
  ];

  const filteredPlaybook =
    selectedPlaybookCategory === 'all'
      ? report.remediation_playbook || []
      : (report.remediation_playbook || []).filter(
          (item) => item.target_category === selectedPlaybookCategory
        );

  return (
    <div className="h-full overflow-y-auto bg-[#08090C] text-[#E1E4EA] p-6 space-y-6">
      {/* Top Banner / Metrics Header */}
      <div className="border border-[#1E222D] bg-[#0D0F14] rounded p-5">
        <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-6">
          {/* Risk Score Gauge */}
          <div className="flex items-center space-x-6">
            <div className={`p-4 rounded border ${riskColor.border} bg-[#08090C] flex flex-col items-center justify-center min-w-[120px]`}>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#8A90A2]">
                RISK SCORE
              </span>
              <div className="flex items-baseline space-x-1 my-1">
                <span className={`text-4xl font-mono font-bold ${riskColor.text}`}>
                  {riskScore}
                </span>
                <span className="text-xs font-mono text-[#8A90A2]">/100</span>
              </div>
              <span className={`text-[10px] font-mono px-2 py-0.5 rounded border ${riskColor.border} font-semibold ${riskColor.text}`}>
                {riskColor.label}
              </span>
            </div>

            <div>
              <div className="flex items-center space-x-2">
                <h2 className="text-base font-mono font-semibold text-[#E1E4EA] tracking-tight">
                  AUTONOMOUS THREAT SYNTHESIS
                </h2>
                {report.is_fallback ? (
                  <span className="px-1.5 py-0.5 text-[10px] font-mono bg-[#EAB308]/10 text-[#EAB308] border border-[#EAB308]/30 rounded">
                    DETERMINISTIC FALLBACK
                  </span>
                ) : (
                  <span className="px-1.5 py-0.5 text-[10px] font-mono bg-[#38BDF8]/10 text-[#38BDF8] border border-[#38BDF8]/30 rounded flex items-center space-x-1">
                    <Sparkles className="w-2.5 h-2.5 mr-1" />
                    <span>GENAI VERIFIED</span>
                  </span>
                )}
              </div>
              <p className="text-xs text-[#8A90A2] mt-1 max-w-xl">
                Synthesized by <span className="font-mono text-[#38BDF8]">{report.model_version}</span> consuming{' '}
                <span className="font-mono text-[#E1E4EA]">{report.tokens_used.toLocaleString()}</span> tokens at{' '}
                <span className="font-mono text-[#8A90A2]">
                  {new Date(report.generated_at).toLocaleString()}
                </span>
              </p>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap items-center gap-2">
            {scanJob && (
              <>
                <a
                  href={api.getExportMarkdownUrl(scanJob.id)}
                  download={`reconpilot-report-${scanJob.id}.md`}
                  className="flex items-center space-x-1.5 px-3 py-1.5 bg-[#13161F] hover:bg-[#1E222D] border border-[#1E222D] text-xs font-mono text-[#E1E4EA] rounded transition-colors"
                >
                  <Download className="w-3.5 h-3.5 text-[#38BDF8]" />
                  <span>EXPORT .MD</span>
                </a>
                <a
                  href={api.getExportJsonUrl(scanJob.id)}
                  download={`reconpilot-report-${scanJob.id}.json`}
                  className="flex items-center space-x-1.5 px-3 py-1.5 bg-[#13161F] hover:bg-[#1E222D] border border-[#1E222D] text-xs font-mono text-[#E1E4EA] rounded transition-colors"
                >
                  <FileText className="w-3.5 h-3.5 text-[#10B981]" />
                  <span>EXPORT .JSON</span>
                </a>
                <button
                  onClick={handleRegenerate}
                  disabled={isGenerating}
                  className="flex items-center space-x-1.5 px-3 py-1.5 bg-[#38BDF8]/10 hover:bg-[#38BDF8]/20 border border-[#38BDF8]/40 text-[#38BDF8] text-xs font-mono rounded transition-colors disabled:opacity-50"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${isGenerating ? 'animate-spin' : ''}`} />
                  <span>{isGenerating ? 'SYNTHESIZING...' : 'RE-RUN AI'}</span>
                </button>
              </>
            )}
          </div>
        </div>

        {/* Calibrated Risk Bar */}
        <div className="mt-4 pt-4 border-t border-[#1E222D]">
          <div className="flex items-center justify-between text-[10px] font-mono text-[#8A90A2] mb-1.5">
            <span>DEFENSIVE SURFACE EXPOSURE CALIBRATION</span>
            <span>{riskScore} / 100</span>
          </div>
          <div className="h-1.5 w-full bg-[#13161F] rounded overflow-hidden flex">
            <div
              className={`h-full ${riskColor.bar} transition-all duration-500`}
              style={{ width: `${Math.min(100, Math.max(0, riskScore))}%` }}
            />
          </div>
        </div>
      </div>

      {/* Executive Summary Section */}
      <div className="border border-[#1E222D] bg-[#0D0F14] rounded p-5">
        <div className="flex items-center space-x-2 mb-3">
          <Terminal className="w-4 h-4 text-[#38BDF8]" />
          <h3 className="text-xs font-mono font-semibold uppercase tracking-wider text-[#E1E4EA]">
            EXECUTIVE POSTURE BRIEFING
          </h3>
        </div>
        <div className="bg-[#08090C] border border-[#1E222D] rounded p-4 font-mono text-xs text-[#8A90A2] leading-relaxed whitespace-pre-line">
          {report.executive_summary}
        </div>
      </div>

      {/* Attack Surface Drift Analysis */}
      {report.attack_surface_drift_analysis && (
        <div className="border border-[#1E222D] bg-[#0D0F14] rounded p-5">
          <div className="flex items-center space-x-2 mb-4">
            <Activity className="w-4 h-4 text-[#EAB308]" />
            <h3 className="text-xs font-mono font-semibold uppercase tracking-wider text-[#E1E4EA]">
              ATTACK SURFACE DRIFT DELTA
            </h3>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-4">
            <div className="border border-[#1E222D] bg-[#08090C] p-3 rounded">
              <span className="text-[10px] font-mono text-[#8A90A2] uppercase block">
                NEW HOSTS DISCOVERED
              </span>
              <span className="text-xl font-mono font-bold text-[#10B981] mt-1 block">
                +{report.attack_surface_drift_analysis.total_new_hosts}
              </span>
            </div>
            <div className="border border-[#1E222D] bg-[#08090C] p-3 rounded">
              <span className="text-[10px] font-mono text-[#8A90A2] uppercase block">
                DECOMMISSIONED HOSTS
              </span>
              <span className="text-xl font-mono font-bold text-[#8A90A2] mt-1 block">
                -{report.attack_surface_drift_analysis.total_removed_hosts}
              </span>
            </div>
            <div className="border border-[#1E222D] bg-[#08090C] p-3 rounded">
              <span className="text-[10px] font-mono text-[#8A90A2] uppercase block">
                NEW OPEN PORTS
              </span>
              <span className="text-xl font-mono font-bold text-[#F97316] mt-1 block">
                +{report.attack_surface_drift_analysis.total_opened_ports}
              </span>
            </div>
            <div className="border border-[#1E222D] bg-[#08090C] p-3 rounded">
              <span className="text-[10px] font-mono text-[#8A90A2] uppercase block">
                PORTS SHUT DOWN
              </span>
              <span className="text-xl font-mono font-bold text-[#10B981] mt-1 block">
                -{report.attack_surface_drift_analysis.total_closed_ports}
              </span>
            </div>
          </div>

          <div className="bg-[#08090C] border border-[#1E222D] rounded p-3 mb-3 text-xs text-[#8A90A2]">
            <p className="font-mono text-[11px] leading-relaxed">
              {report.attack_surface_drift_analysis.summary_description}
            </p>
          </div>

          {report.attack_surface_drift_analysis.high_risk_mutations &&
            report.attack_surface_drift_analysis.high_risk_mutations.length > 0 && (
              <div className="space-y-1.5">
                <span className="text-[10px] font-mono uppercase text-[#F43F5E] font-semibold">
                  HIGH-RISK SURFACE MUTATIONS
                </span>
                <div className="space-y-1">
                  {report.attack_surface_drift_analysis.high_risk_mutations.map((mut, idx) => (
                    <div
                      key={idx}
                      className="flex items-center space-x-2 text-xs font-mono bg-[#2A0C14] border border-[#F43F5E]/30 text-[#F43F5E] px-3 py-1.5 rounded"
                    >
                      <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
                      <span>{mut}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
        </div>
      )}

      {/* Prioritized Findings Matrix */}
      {report.prioritized_findings && report.prioritized_findings.length > 0 && (
        <div className="border border-[#1E222D] bg-[#0D0F14] rounded p-5">
          <div className="flex items-center space-x-2 mb-4">
            <Layers className="w-4 h-4 text-[#F43F5E]" />
            <h3 className="text-xs font-mono font-semibold uppercase tracking-wider text-[#E1E4EA]">
              SYNTHESIZED EXPLOITABILITY RANKING
            </h3>
          </div>

          <div className="space-y-3">
            {report.prioritized_findings.map((item, idx) => {
              const sevBadge =
                item.severity === 'critical'
                  ? 'bg-[#2A0C14] border-[#F43F5E]/40 text-[#F43F5E]'
                  : item.severity === 'high'
                  ? 'bg-[#2E1508] border-[#F97316]/40 text-[#F97316]'
                  : 'bg-[#291F08] border-[#EAB308]/40 text-[#EAB308]';

              return (
                <div
                  key={idx}
                  className="border border-[#1E222D] bg-[#08090C] rounded p-4 hover:border-[#38BDF8]/40 transition-colors"
                >
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-2">
                    <div className="flex items-center space-x-2.5">
                      <span className="text-xs font-mono font-bold text-[#8A90A2] px-1.5 py-0.5 bg-[#13161F] border border-[#1E222D] rounded">
                        #{item.rank ?? idx + 1}
                      </span>
                      <h4 className="text-xs font-mono font-semibold text-[#E1E4EA]">
                        {item.title}
                      </h4>
                      <span className={`text-[10px] font-mono uppercase px-2 py-0.5 border rounded ${sevBadge}`}>
                        {item.severity}
                      </span>
                    </div>
                    <span className="text-xs font-mono text-[#38BDF8] bg-[#13161F] px-2 py-0.5 rounded border border-[#1E222D]">
                      {item.affected_asset}
                    </span>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs mt-3">
                    <div className="bg-[#0D0F14] border border-[#1E222D] p-2.5 rounded">
                      <span className="text-[10px] font-mono text-[#8A90A2] block uppercase mb-1">
                        PROBABILITY OF EXPLOIT
                      </span>
                      <p className="font-mono text-[#E1E4EA] text-[11px]">
                        {item.exploit_probability}
                      </p>
                    </div>

                    <div className="bg-[#0D0F14] border border-[#1E222D] p-2.5 rounded">
                      <span className="text-[10px] font-mono text-[#8A90A2] block uppercase mb-1">
                        BUSINESS IMPACT
                      </span>
                      <p className="font-mono text-[#E1E4EA] text-[11px]">
                        {item.impact_analysis}
                      </p>
                    </div>

                    <div className="bg-[#0D0F14] border border-[#1E222D] p-2.5 rounded">
                      <span className="text-[10px] font-mono text-[#8A90A2] block uppercase mb-1">
                        ROOT CAUSE ANALYSIS
                      </span>
                      <p className="font-mono text-[#8A90A2] text-[11px]">
                        {item.root_cause}
                      </p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Defensive Remediation Playbook */}
      {report.remediation_playbook && report.remediation_playbook.length > 0 && (
        <div className="border border-[#1E222D] bg-[#0D0F14] rounded p-5">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
            <div className="flex items-center space-x-2">
              <ShieldCheck className="w-4 h-4 text-[#10B981]" />
              <h3 className="text-xs font-mono font-semibold uppercase tracking-wider text-[#E1E4EA]">
                HARDENING PLAYBOOK & VALIDATION
              </h3>
            </div>

            {/* Category Filter Pills */}
            <div className="flex flex-wrap items-center gap-1.5">
              {categories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setSelectedPlaybookCategory(cat)}
                  className={`px-2 py-0.5 text-[10px] font-mono uppercase rounded border transition-colors ${
                    selectedPlaybookCategory === cat
                      ? 'bg-[#38BDF8]/10 border-[#38BDF8] text-[#38BDF8]'
                      : 'bg-[#08090C] border-[#1E222D] text-[#8A90A2] hover:text-[#E1E4EA]'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>

          <div className="space-y-4">
            {filteredPlaybook.map((item, idx) => (
              <div key={idx} className="border border-[#1E222D] bg-[#08090C] rounded p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <span className="px-1.5 py-0.5 text-[10px] font-mono uppercase bg-[#13161F] border border-[#1E222D] text-[#38BDF8] rounded">
                      {item.target_category}
                    </span>
                    <h4 className="text-xs font-mono font-semibold text-[#E1E4EA]">
                      {item.title}
                    </h4>
                  </div>
                  <span className="text-[10px] font-mono uppercase text-[#F97316] bg-[#2E1508] border border-[#F97316]/30 px-1.5 py-0.5 rounded">
                    {item.severity}
                  </span>
                </div>

                <p className="text-xs font-mono text-[#8A90A2] mb-3">
                  {item.explanation}
                </p>

                {/* Copyable Code Snippet */}
                {item.config_snippet && (
                  <div className="mb-3 relative group">
                    <div className="flex items-center justify-between bg-[#13161F] border-t border-x border-[#1E222D] px-3 py-1.5 text-[10px] font-mono text-[#8A90A2]">
                      <span>PATCH SNIPPET</span>
                      <button
                        onClick={() => handleCopySnippet(item.config_snippet, idx)}
                        className="flex items-center space-x-1 text-[#38BDF8] hover:text-[#E1E4EA] transition-colors"
                      >
                        {copiedSnippetIndex === idx ? (
                          <>
                            <Check className="w-3 h-3 text-[#10B981]" />
                            <span className="text-[#10B981]">COPIED</span>
                          </>
                        ) : (
                          <>
                            <Copy className="w-3 h-3" />
                            <span>COPY CODE</span>
                          </>
                        )}
                      </button>
                    </div>
                    <pre className="p-3 bg-[#0D0F14] border border-[#1E222D] text-xs font-mono text-[#10B981] overflow-x-auto selection:bg-[#38BDF8]/20">
                      <code>{item.config_snippet}</code>
                    </pre>
                  </div>
                )}

                {/* Validation Steps */}
                {item.validation_steps && item.validation_steps.length > 0 && (
                  <div className="border border-[#1E222D] bg-[#0D0F14] p-3 rounded">
                    <span className="text-[10px] font-mono uppercase text-[#8A90A2] block mb-2 font-semibold">
                      VERIFICATION CHECKLIST
                    </span>
                    <ul className="space-y-1.5">
                      {item.validation_steps.map((step, sIdx) => (
                        <li key={sIdx} className="flex items-start space-x-2 text-xs font-mono text-[#8A90A2]">
                          <span className="text-[#38BDF8] mt-0.5 font-bold">›</span>
                          <span className="text-[#E1E4EA]">{step}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

