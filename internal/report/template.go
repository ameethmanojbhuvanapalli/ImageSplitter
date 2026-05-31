package report

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Image Splitter &#8212; Run {{runPad .RunResult.RunNumber}}</title>
<style>
  :root {
    --bg:#f5f7fa;--surface:#fff;--border:#e2e8f0;--text:#1a202c;--muted:#718096;
    --accent:#4f46e5;--green:#22c55e;--green-bg:#f0fdf4;--green-bd:#bbf7d0;
    --yellow:#ca8a04;--yellow-bg:#fefce8;--yellow-bd:#fde68a;
    --red:#dc2626;--red-bg:#fef2f2;--red-bd:#fecaca;
    --radius:8px;--shadow:0 1px 3px rgba(0,0,0,.08),0 1px 2px rgba(0,0,0,.04);
  }
  *{box-sizing:border-box;margin:0;padding:0}
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:var(--bg);color:var(--text);font-size:14px;line-height:1.5}
  header{background:var(--surface);border-bottom:1px solid var(--border);padding:20px 32px;display:flex;align-items:center;gap:16px}
  .logo{font-size:22px;font-weight:700;color:var(--accent)}
  .run-badge{background:var(--accent);color:#fff;padding:3px 12px;border-radius:20px;font-size:13px;font-weight:600}
  .main{padding:28px 32px;max-width:1400px;margin:0 auto}
  .grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:16px;margin-bottom:28px}
  .card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:18px 20px;box-shadow:var(--shadow)}
  .card .label{font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.06em;color:var(--muted);margin-bottom:6px}
  .card .value{font-size:26px;font-weight:700;line-height:1.1}
  .card .sub{font-size:12px;color:var(--muted);margin-top:3px}
  .card.green{border-left:4px solid var(--green)}.card.green .value{color:#16a34a}
  .card.yellow{border-left:4px solid var(--yellow)}.card.yellow .value{color:var(--yellow)}
  .card.red{border-left:4px solid var(--red)}.card.red .value{color:var(--red)}
  .meta{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:16px 20px;display:flex;flex-wrap:wrap;gap:24px;margin-bottom:24px;box-shadow:var(--shadow)}
  .meta-label{font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.06em;color:var(--muted)}
  .meta-value{font-size:14px;font-weight:500;margin-top:2px}
  .controls{display:flex;gap:12px;margin-bottom:16px;flex-wrap:wrap;align-items:center}
  .search{flex:1;min-width:220px;padding:9px 14px;border:1px solid var(--border);border-radius:var(--radius);font-size:14px;background:var(--surface);outline:none;transition:border-color .15s}
  .search:focus{border-color:var(--accent)}
  .fbtn{padding:8px 16px;border-radius:var(--radius);border:1px solid var(--border);background:var(--surface);font-size:13px;font-weight:500;cursor:pointer;transition:all .15s;white-space:nowrap}
  .fa{background:var(--accent);color:#fff;border-color:var(--accent)}
  .fg{background:#16a34a;color:#fff;border-color:#16a34a}
  .fy{background:var(--yellow);color:#fff;border-color:var(--yellow)}
  .fr2{background:var(--red);color:#fff;border-color:var(--red)}
  .rcount{font-size:13px;color:var(--muted);margin-left:auto;align-self:center}
  .table-wrap{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);box-shadow:var(--shadow);overflow:hidden}
  table{width:100%;border-collapse:collapse}
  thead th{background:#f8fafc;padding:12px 16px;text-align:left;font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:.06em;color:var(--muted);border-bottom:1px solid var(--border);cursor:pointer;user-select:none;white-space:nowrap}
  thead th:hover{background:#edf2f7;color:var(--text)}
  th.asc::after{content:" \2191"}th.desc::after{content:" \2193"}
  tbody tr.folder-row{border-bottom:1px solid var(--border);cursor:pointer;transition:background .1s}
  tbody tr.folder-row:hover{background:#f8fafc}
  tbody tr.image-row{border-bottom:1px solid var(--border);background:#fafafa;display:none}
  tbody tr.image-row td:first-child{padding-left:36px;font-family:"SFMono-Regular",Consolas,monospace;font-size:12px;color:var(--muted)}
  tbody tr.folder-row.expanded + tr.image-row,
  tbody tr.image-row + tr.image-row { display: table-row; }
  td{padding:11px 16px;vertical-align:middle}
  .col-name{font-weight:600;font-size:13.5px}
  .col-path{font-family:"SFMono-Regular",Consolas,monospace;font-size:12px;color:var(--muted);word-break:break-all;max-width:380px}
  .col-det{color:var(--muted);font-size:13px}
  .status-error .col-det{color:#b91c1c;font-weight:500}
  .badge{display:inline-block;padding:3px 10px;border-radius:12px;font-size:12px;font-weight:600;white-space:nowrap}
  .status-processed .badge{background:var(--green-bg);color:#15803d;border:1px solid var(--green-bd)}
  .status-already   .badge{background:var(--yellow-bg);color:#a16207;border:1px solid var(--yellow-bd)}
  .status-missing   .badge{background:var(--yellow-bg);color:#a16207;border:1px solid var(--yellow-bd)}
  .status-error     .badge{background:var(--red-bg);color:#b91c1c;border:1px solid var(--red-bd)}
  .expand-icon{font-size:10px;margin-right:6px;display:inline-block;transition:transform .2s}
  .expanded .expand-icon{transform:rotate(90deg)}
  .empty{text-align:center;padding:48px 24px;color:var(--muted)}
  footer{text-align:center;padding:24px;color:var(--muted);font-size:12px;border-top:1px solid var(--border);margin-top:32px}
</style>
</head>
<body>
<header>
  <div class="logo">&#9988; Image Splitter</div>
  <div class="run-badge">Run {{runPad .RunResult.RunNumber}}</div>
</header>
<div class="main">
  <div class="meta">
    <div><div class="meta-label">Date</div><div class="meta-value">{{formatTime .RunResult.StartTime}}</div></div>
    <div><div class="meta-label">Duration</div><div class="meta-value">{{formatDuration .RunResult.Duration}}</div></div>
    <div><div class="meta-label">Folders Scanned</div><div class="meta-value">{{.RunResult.TotalFolders}}</div></div>
  </div>
  <div class="grid">
    <div class="card green"><div class="label">Images Split</div><div class="value">{{.ProcessedCount}}</div><div class="sub">Split successfully</div></div>
    <div class="card yellow"><div class="label">Already Done</div><div class="value">{{.AlreadyProcessed}}</div><div class="sub">Output existed</div></div>
    <div class="card yellow"><div class="label">Image Missing</div><div class="value">{{.MissingCount}}</div><div class="sub">File not found</div></div>
    <div class="card red"><div class="label">Errors</div><div class="value">{{.ErrorCount}}</div><div class="sub">Unexpected failures</div></div>
  </div>
  <div class="controls">
    <input type="text" class="search" id="si" placeholder="Search folders&#8230;" oninput="af()"/>
    <button class="fbtn fa" id="ba" onclick="sf('a')">All ({{.RunResult.TotalFolders}})</button>
    <button class="fbtn"    id="bg" onclick="sf('g')">&#10003; Processed ({{.ProcessedCount}})</button>
    <button class="fbtn"    id="by" onclick="sf('y')">&#9888; Skipped ({{add .AlreadyProcessed .MissingCount}})</button>
    <button class="fbtn"    id="br" onclick="sf('r')">&#10007; Errors ({{.ErrorCount}})</button>
    <span class="rcount" id="rc"></span>
  </div>
  <div class="table-wrap">
    <table>
      <thead><tr>
        <th onclick="st(0)">Folder Name</th>
        <th onclick="st(1)">Folder Path</th>
        <th onclick="st(2)">Status</th>
        <th onclick="st(3)">Details</th>
      </tr></thead>
      <tbody id="tb">
        {{range .RunResult.FolderResults}}
        <tr class="folder-row {{overallClass .}}" data-s="{{overallClass .}}" onclick="toggle(this)">
          <td class="col-name"><span class="expand-icon">&#9658;</span>{{.FolderName}}</td>
          <td class="col-path">{{.FolderPath}}</td>
          <td><span class="badge">{{overallLabel .}}</span></td>
          <td class="col-det">{{len .ImageResults}} image(s) checked</td>
        </tr>
        {{range .ImageResults}}
        <tr class="image-row {{statusClass .Status}}">
          <td>&#8627; {{.FileName}}</td>
          <td></td>
          <td><span class="badge">{{statusLabel .Status}}</span></td>
          <td class="col-det">{{.Message}}</td>
        </tr>
        {{end}}
        {{end}}
      </tbody>
    </table>
    <div class="empty" id="es" style="display:none"><p>&#128269; No results match your search or filter.</p></div>
  </div>
</div>
<footer>Image Splitter &mdash; Run {{runPad .RunResult.RunNumber}} &mdash; Generated {{formatTime .RunResult.EndTime}}</footer>
<script>
(function(){
  var cf='a',sc=-1,sa=true;
  var fm={a:null,g:['status-processed'],y:['status-already','status-missing'],r:['status-error']};
  var ac={a:'fa',g:'fg',y:'fy',r:'fr2'};

  window.toggle=function(row){
    row.classList.toggle('expanded');
    // show/hide immediately following image-rows
    var next=row.nextElementSibling;
    while(next && next.classList.contains('image-row')){
      next.style.display=row.classList.contains('expanded')?'table-row':'none';
      next=next.nextElementSibling;
    }
  };

  window.sf=function(f){
    cf=f;
    ['a','g','y','r'].forEach(function(k){
      var b=document.getElementById('b'+k);
      b.className='fbtn';
      if(k===f)b.classList.add(ac[f]);
    });
    af();
  };

  window.af=function(){
    var s=document.getElementById('si').value.toLowerCase();
    var al=fm[cf];
    var rows=document.querySelectorAll('#tb tr.folder-row');
    var v=0;
    rows.forEach(function(row){
      var ok=(!al||al.indexOf(row.dataset.s)!==-1)&&(!s||row.textContent.toLowerCase().indexOf(s)!==-1);
      row.style.display=ok?'':'none';
      // also hide/show child image rows
      var next=row.nextElementSibling;
      while(next&&next.classList.contains('image-row')){
        next.style.display=(ok&&row.classList.contains('expanded'))?'table-row':'none';
        next=next.nextElementSibling;
      }
      if(ok)v++;
    });
    var t=rows.length;
    document.getElementById('rc').textContent=v===t?'':'Showing '+v+' of '+t;
    document.getElementById('es').style.display=v===0?'':'none';
  };

  window.st=function(col){
    var tb=document.getElementById('tb');
    var folders=[];
    var rows=Array.prototype.slice.call(tb.querySelectorAll('tr'));
    // Group folder rows with their image rows
    var current=null;
    rows.forEach(function(r){
      if(r.classList.contains('folder-row')){
        current={folder:r,images:[]};
        folders.push(current);
      } else if(current){
        current.images.push(r);
      }
    });
    if(sc===col){sa=!sa;}else{sc=col;sa=true;}
    document.querySelectorAll('thead th').forEach(function(th,i){
      th.classList.remove('asc','desc');
      if(i===col)th.classList.add(sa?'asc':'desc');
    });
    folders.sort(function(a,b){
      var at=a.folder.cells[col].textContent.trim().toLowerCase();
      var bt=b.folder.cells[col].textContent.trim().toLowerCase();
      return at<bt?(sa?-1:1):at>bt?(sa?1:-1):0;
    });
    folders.forEach(function(g){
      tb.appendChild(g.folder);
      g.images.forEach(function(r){tb.appendChild(r);});
    });
    af();
  };

  af();
})();
</script>
</body>
</html>`
