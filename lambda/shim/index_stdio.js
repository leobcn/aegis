
const MAX_FAILS = 4;

process.env['PATH'] = process.env['PATH'] + ':' + process.env['LAMBDA_TASK_ROOT'] + ':' + __dirname;
// console.log(process.env['PATH']);

var child_process = require('child_process'),
  go_proc = null,
  done = console.log.bind(console),
  fails = 0;

// TODO: Try to avoid this? I can't imagine copying, not moving, this app over each time is great for performance.
// To avoid permission issues...
// Really hate to copy this file each time (and mv didn't work).
child_process.execSync('cp aegis_app /tmp/aegis_app && chmod +x /tmp/aegis_app'); 

// Debug
// var fs = require('fs');
// items = fs.readdirSync('./');
// for (var i=0; i<items.length; i++) {
//     console.log("");
//     console.log("");
//     console.log("FILE: " + items[i]);
//     var stats = fs.statSync(items[i]);
//     console.log("------------------------");
//     console.log(stats);
//     console.log();
 
//     if (stats.isFile()) {
//         console.log('    file');
//     }
//     if (stats.isDirectory()) {
//         console.log('    directory');
//     }
 
//     console.log('    size: ' + stats["size"]);
//     console.log('    mode: ' + stats["mode"]);
// }

(function new_go_proc() {
  // pipe stdin/out, blind passthru stderr
  go_proc = child_process.spawn('/tmp/aegis_app', { stdio: ['pipe', 'pipe', process.stderr] });
  
  //child_process.execSync('chmod +x aegis_app'); // can't do this, operation not permitted
  // go_proc = child_process.spawn('./aegis_app', { stdio: ['pipe', 'pipe', process.stderr] }); // this used to work, why not now? ¯\_(ツ)_/¯

  go_proc.on('error', function(err) {
    process.stderr.write("go_proc errored: "+JSON.stringify(err)+"\n");
    if (++fails > MAX_FAILS) {
      process.exit(1); // force container restart after too many fails
    }
    new_go_proc();
    done(err);
  });

  go_proc.on('exit', function(code) {
    process.stderr.write("go_proc exited prematurely with code: "+code+"\n");
    if (++fails > MAX_FAILS) {
      process.exit(1); // force container restart after too many fails
    }
    new_go_proc();
    done(new Error("Exited with code "+code));
  });

  go_proc.stdin.on('error', function(err) {
    process.stderr.write("go_proc stdin write error: "+JSON.stringify(err)+"\n");
    if (++fails > MAX_FAILS) {
      process.exit(1); // force container restart after too many fails
    }
    new_go_proc();
    done(err);
  });

  var data = null;
  go_proc.stdout.on('data', function(chunk) {
    fails = 0; // reset fails
    if (data === null) {
      data = new Buffer(chunk);
    } else {
      data.write(chunk);
    }
    // check for newline ascii char 10
    if (data.length && data[data.length-1] == 10) {
      var output = JSON.parse(data.toString('UTF-8'));
      data = null;
      done(null, output);
    };
  });
})();

exports.handler = function(event, context) {
  // add to event the invoke time (oddly not present in context or event)
  // I wish it was from when API Gateway received the request...not sure if there's a way to pass that info.
  var hrTime = process.hrtime();
  event.handlerStartHrTime = hrTime;
  // and normal JavaScript milliseconds
  event.handlerStartTimeMs = new Date().getTime();

  // always output to current context's done
  done = context.done.bind(context);

  go_proc.stdin.write(JSON.stringify({
    "event": event,
    "context": context
  })+"\n");

}