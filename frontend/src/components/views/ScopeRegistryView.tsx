import React, { useState } from 'react';
import { Scope } from '../../types';
import { api } from '../../api/client';
import {
  ShieldCheck,
  ShieldAlert,
  Clock,
  Plus,
  Trash2,
  Check,
  Copy,
  ExternalLink,
  RefreshCw,
  Sliders,
  Globe,
  AlertTriangle,
  X,
} from 'lucide-react';

interface ScopeRegistryViewProps {
  scopes: Scope[];
  onScopesUpdated: () => void;
  onSelectScopeToScan?: (scope: Scope) => void;
}

export const ScopeRegistryView: React.FC<ScopeRegistryViewProps> = ({
  scopes,
  onScopesUpdated,
  onSelectScopeToScan,
}) => {
  const [isRegisterOpen, setIsRegisterOpen] = useState(false);
  const [newDomain, setNewDomain] = useState('');
  const [newName, setNewName] = useState('');
  const [newRateLimit, setNewRateLimit] = useState(25);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [verifyingId, setVerifyingId] = useState<string | null>(null);
  const [verificationFeedback, setVerificationFeedback] = useState<string | null>(null);
  const [copiedTokenId, setCopiedTokenId] = useState<string | null>(null);

  const handleRegisterScope = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomain.trim()) return;

    setIsSubmitting(true);
    try {
      await api.createScope({
        name: newName.trim() || newDomain.trim(),
        root_domain: newDomain.trim().toLowerCase(),
        rate_limit_rps: Number(newRateLimit) || 25,
      });
      setIsRegisterOpen(false);
      setNewDomain('');
      setNewName('');
      onScopesUpdated();
    } catch (err: any) {
      alert(`Scope creation failed: ${err.message}`);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleVerify = async (scope: Scope, mock = false) => {
    setVerifyingId(scope.id);
    setVerificationFeedback(null);
    try {
      const res = await api.verifyScope(scope.id, mock);
      setVerificationFeedback(res.details || (res.verified ? 'Verification confirmed.' : 'Verification failed.'));
      onScopesUpdated();
    } catch (err: any) {
      setVerificationFeedback(`Verification check failed: ${err.message}`);
    } finally {
      setVerifyingId(null);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to remove scope "${name}"? This will deregister all associated targets.`)) {
      return;
    }
    try {
      await api.deleteScope(id);
      onScopesUpdated();
    } catch (err: any) {
      alert(`Failed to delete scope: ${err.message}`);
    }
  };

  const handleCopyToken = (token: string, id: string) => {
    navigator.clipboard.writeText(token);
    setCopiedTokenId(id);
    setTimeout(() => setCopiedTokenId(null), 2000);
  };

  return (
    <div className="h-full overflow-y-auto bg-[#08090C] text-[#E1E4EA] p-6 space-y-6">
      {/* Top Header & Metrics */}
      <div className="border border-[#1E222D] bg-[#0D0F14] rounded p-5">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <div className="flex items-center space-x-2">
              <Globe className="w-4 h-4 text-[#38BDF8]" />
              <h2 className="text-sm font-mono font-semibold uppercase tracking-wider text-[#E1E4EA]">
                TARGET SCOPE & DNS OWNERSHIP REGISTRY
              </h2>
            </div>
            <p className="text-xs font-mono text-[#8A90A2] mt-1">
              Strict boundary enforcement. ReconPilot only executes active scans against domains
              verified via DNS TXT challenge records (<code className="text-[#38BDF8]">_reconpilot-challenge</code>).
            </p>
          </div>

          <button
            onClick={() => setIsRegisterOpen(true)}
            className="flex items-center space-x-1.5 px-3 py-1.5 bg-[#38BDF8] hover:bg-[#38BDF8]/90 text-[#08090C] text-xs font-mono font-semibold rounded transition-colors self-start sm:self-auto"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>ENROLL ROOT DOMAIN</span>
          </button>
        </div>

        {verificationFeedback && (
          <div className="mt-4 p-3 bg-[#13161F] border border-[#38BDF8]/30 rounded text-xs font-mono text-[#38BDF8] flex items-center justify-between">
            <span>{verificationFeedback}</span>
            <button
              onClick={() => setVerificationFeedback(null)}
              className="text-[#8A90A2] hover:text-[#E1E4EA]"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}
      </div>

      {/* Scope Table */}
      <div className="border border-[#1E222D] bg-[#0D0F14] rounded overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-[#1E222D] bg-[#08090C] text-[10px] font-mono text-[#8A90A2] uppercase tracking-wider">
                <th className="py-2.5 px-4 font-medium">ORGANIZATION / SCOPE</th>
                <th className="py-2.5 px-4 font-medium">ROOT DOMAIN</th>
                <th className="py-2.5 px-4 font-medium">OWNERSHIP STATUS</th>
                <th className="py-2.5 px-4 font-medium">DNS TXT CHALLENGE TOKEN</th>
                <th className="py-2.5 px-4 font-medium">RATE LIMIT</th>
                <th className="py-2.5 px-4 font-medium text-right">ACTIONS</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#1E222D] text-xs font-mono">
              {scopes.map((scope) => {
                const isVerified = scope.verification_status === 'verified';
                const isPending = scope.verification_status === 'pending_verification';

                return (
                  <tr key={scope.id} className="hover:bg-[#13161F]/60 transition-colors">
                    {/* Organization / Scope Name */}
                    <td className="py-3 px-4">
                      <div className="font-semibold text-[#E1E4EA]">{scope.name}</div>
                      <div className="text-[10px] text-[#8A90A2]">ID: {scope.id}</div>
                    </td>

                    {/* Root Domain */}
                    <td className="py-3 px-4">
                      <span className="text-[#38BDF8] font-semibold">{scope.root_domain}</span>
                    </td>

                    {/* Status Badge */}
                    <td className="py-3 px-4">
                      {isVerified ? (
                        <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded text-[10px] font-mono bg-[#10B981]/10 text-[#10B981] border border-[#10B981]/30">
                          <ShieldCheck className="w-3 h-3" />
                          <span>VERIFIED</span>
                        </span>
                      ) : isPending ? (
                        <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded text-[10px] font-mono bg-[#EAB308]/10 text-[#EAB308] border border-[#EAB308]/30">
                          <Clock className="w-3 h-3" />
                          <span>TXT PENDING</span>
                        </span>
                      ) : (
                        <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded text-[10px] font-mono bg-[#F43F5E]/10 text-[#F43F5E] border border-[#F43F5E]/30">
                          <ShieldAlert className="w-3 h-3" />
                          <span>REVOKED</span>
                        </span>
                      )}
                    </td>

                    {/* Verification Token */}
                    <td className="py-3 px-4">
                      <div className="flex items-center space-x-2">
                        <code className="text-[10px] text-[#8A90A2] bg-[#08090C] border border-[#1E222D] px-2 py-1 rounded max-w-[220px] truncate select-all">
                          {scope.verification_token}
                        </code>
                        <button
                          onClick={() => handleCopyToken(scope.verification_token, scope.id)}
                          title="Copy verification challenge token"
                          className="p-1 hover:text-[#38BDF8] text-[#8A90A2] transition-colors"
                        >
                          {copiedTokenId === scope.id ? (
                            <Check className="w-3.5 h-3.5 text-[#10B981]" />
                          ) : (
                            <Copy className="w-3.5 h-3.5" />
                          )}
                        </button>
                      </div>
                    </td>

                    {/* Rate Limit */}
                    <td className="py-3 px-4 text-[#8A90A2]">
                      <span className="text-[#E1E4EA] font-semibold">{scope.rate_limit_rps}</span> RPS
                    </td>

                    {/* Actions */}
                    <td className="py-3 px-4 text-right">
                      <div className="flex items-center justify-end space-x-2">
                        {!isVerified && (
                          <button
                            onClick={() => handleVerify(scope, false)}
                            disabled={verifyingId === scope.id}
                            className="px-2 py-1 bg-[#13161F] hover:bg-[#1E222D] border border-[#1E222D] text-[10px] text-[#38BDF8] rounded transition-colors disabled:opacity-50"
                          >
                            {verifyingId === scope.id ? 'CHECKING...' : 'VERIFY DNS'}
                          </button>
                        )}
                        {onSelectScopeToScan && isVerified && (
                          <button
                            onClick={() => onSelectScopeToScan(scope)}
                            className="px-2 py-1 bg-[#38BDF8]/10 hover:bg-[#38BDF8]/20 border border-[#38BDF8]/40 text-[10px] text-[#38BDF8] rounded transition-colors"
                          >
                            NEW SCAN
                          </button>
                        )}
                        <button
                          onClick={() => handleDelete(scope.id, scope.name)}
                          className="p-1 text-[#8A90A2] hover:text-[#F43F5E] transition-colors"
                          title="Delete Scope"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* DNS TXT Setup Guide Card */}
      <div className="border border-[#1E222D] bg-[#0D0F14] rounded p-5">
        <h3 className="text-xs font-mono font-semibold uppercase text-[#E1E4EA] mb-2 flex items-center space-x-2">
          <Sliders className="w-4 h-4 text-[#38BDF8]" />
          <span>AUTHORIZATION & OWNERSHIP VERIFICATION INSTRUCTIONS</span>
        </h3>
        <p className="text-xs font-mono text-[#8A90A2] mb-3">
          To prove administrative control over the target infrastructure before active probes execute:
        </p>
        <div className="bg-[#08090C] border border-[#1E222D] rounded p-3 text-xs font-mono text-[#8A90A2] space-y-2">
          <div className="flex items-start space-x-2">
            <span className="text-[#38BDF8] font-bold">1.</span>
            <span>
              Log into your DNS provider (Cloudflare, Route53, Namecheap, etc.) for the enrolled domain.
            </span>
          </div>
          <div className="flex items-start space-x-2">
            <span className="text-[#38BDF8] font-bold">2.</span>
            <span>
              Create a new <code className="text-[#E1E4EA]">TXT</code> record at hostname{' '}
              <code className="text-[#38BDF8]">_reconpilot-challenge.&lt;domain&gt;</code>
            </span>
          </div>
          <div className="flex items-start space-x-2">
            <span className="text-[#38BDF8] font-bold">3.</span>
            <span>
              Set the value to the unique verification token shown in the table above and click{' '}
              <code className="text-[#E1E4EA]">VERIFY DNS</code>.
            </span>
          </div>
        </div>
      </div>

      {/* Modal: Enroll Root Domain */}
      {isRegisterOpen && (
        <div className="fixed inset-0 z-50 bg-[#08090C]/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="border border-[#1E222D] bg-[#0D0F14] w-full max-w-md rounded shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-100">
            <div className="flex items-center justify-between px-4 py-3 border-b border-[#1E222D] bg-[#08090C]">
              <div className="flex items-center space-x-2">
                <Globe className="w-4 h-4 text-[#38BDF8]" />
                <h3 className="text-xs font-mono font-bold uppercase text-[#E1E4EA]">
                  ENROLL NEW ROOT DOMAIN
                </h3>
              </div>
              <button
                onClick={() => setIsRegisterOpen(false)}
                className="text-[#8A90A2] hover:text-[#E1E4EA]"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleRegisterScope} className="p-4 space-y-4">
              <div>
                <label className="text-[10px] font-mono text-[#8A90A2] uppercase block mb-1">
                  Scope Display Name
                </label>
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="e.g. Acme Production"
                  className="w-full bg-[#08090C] border border-[#1E222D] rounded px-3 py-1.5 text-xs font-mono text-[#E1E4EA] focus:outline-none focus:border-[#38BDF8]"
                />
              </div>

              <div>
                <label className="text-[10px] font-mono text-[#8A90A2] uppercase block mb-1">
                  Root Domain (FQDN) <span className="text-[#F43F5E]">*</span>
                </label>
                <input
                  type="text"
                  required
                  value={newDomain}
                  onChange={(e) => setNewDomain(e.target.value)}
                  placeholder="acme.org"
                  className="w-full bg-[#08090C] border border-[#1E222D] rounded px-3 py-1.5 text-xs font-mono text-[#E1E4EA] focus:outline-none focus:border-[#38BDF8]"
                />
              </div>

              <div>
                <label className="text-[10px] font-mono text-[#8A90A2] uppercase block mb-1">
                  Rate Limit (Requests / Sec)
                </label>
                <input
                  type="number"
                  min="1"
                  max="500"
                  value={newRateLimit}
                  onChange={(e) => setNewRateLimit(Number(e.target.value))}
                  className="w-full bg-[#08090C] border border-[#1E222D] rounded px-3 py-1.5 text-xs font-mono text-[#E1E4EA] focus:outline-none focus:border-[#38BDF8]"
                />
              </div>

              <div className="pt-2 flex items-center justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => setIsRegisterOpen(false)}
                  className="px-3 py-1.5 bg-[#13161F] hover:bg-[#1E222D] border border-[#1E222D] text-xs font-mono text-[#8A90A2] rounded transition-colors"
                >
                  CANCEL
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting || !newDomain.trim()}
                  className="px-3 py-1.5 bg-[#38BDF8] hover:bg-[#38BDF8]/90 text-[#08090C] text-xs font-mono font-bold rounded transition-colors disabled:opacity-50"
                >
                  {isSubmitting ? 'ENROLLING...' : 'REGISTER SCOPE'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

